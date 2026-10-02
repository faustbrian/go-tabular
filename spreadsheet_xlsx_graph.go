package tabular

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

// decodeXLSXDocument consumes the entire part, rather than accepting a valid
// prefix followed by another document or malformed XML.
func decodeXLSXDocument(reader io.Reader, value any) error {
	decoder := xml.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch item := token.(type) {
		case xml.StartElement:
			if err = decoder.DecodeElement(value, &item); err != nil {
				return err
			}
			goto trailing
		case xml.CharData:
			if strings.Trim(string(item), " \t\r\n") != "" {
				return errors.New("unexpected XML content")
			}
		case xml.Comment, xml.ProcInst:
		default:
			return errors.New("unexpected XML content")
		}
	}
trailing:
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		switch item := token.(type) {
		case xml.CharData:
			if strings.Trim(string(item), " \t\r\n") != "" {
				return errors.New("unexpected XML content")
			}
		case xml.Comment:
		default:
			return errors.New("unexpected XML content")
		}
	}
}

func decodeXLSXPart(archive *ZIPArchive, name string, value any) error {
	reader, err := archive.Open(name)
	if err != nil {
		return xlsxPresenceError(err)
	}
	defer func() { _ = reader.Close() }()
	if err = decodeXLSXDocument(reader, value); err != nil {
		return xlsxPresenceError(err)
	}
	return nil
}

// The supported workbook location is canonical. This ensures Excelize and the
// admission/presence readers interpret the same graph before dependency parsing.
func validateXLSXGraph(archive *ZIPArchive, sheet string) error {
	var root struct {
		XMLName xml.Name `xml:"Relationships"`
		Entries []struct {
			ID     string `xml:"Id,attr"`
			Type   string `xml:"Type,attr"`
			Target string `xml:"Target,attr"`
			Mode   string `xml:"TargetMode,attr"`
		} `xml:"Relationship"`
	}
	if err := decodeXLSXPart(archive, "_rels/.rels", &root); err != nil {
		return err
	}
	ids := make(map[string]bool)
	workbooks := 0
	for _, rel := range root.Entries {
		if rel.ID == "" || ids[rel.ID] {
			return xlsxPresenceError(errors.New("root relationships are ambiguous"))
		}
		ids[rel.ID] = true
		if strings.HasSuffix(rel.Type, "/officeDocument") {
			workbooks++
			if strings.TrimPrefix(rel.Target, "/") != "xl/workbook.xml" || rel.Mode != "" || rel.Type != "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" {
				return xlsxPresenceError(errors.New("workbook location is unsupported"))
			}
		}
	}
	if workbooks != 1 {
		return xlsxPresenceError(errors.New("workbook relationships are ambiguous"))
	}
	var content struct {
		XMLName xml.Name `xml:"Types"`
		Entries []struct {
			Part string `xml:"PartName,attr"`
			Type string `xml:"ContentType,attr"`
		} `xml:"Override"`
	}
	if err := decodeXLSXPart(archive, "[Content_Types].xml", &content); err != nil {
		return err
	}
	parts := make(map[string]bool)
	workbooks = 0
	for _, entry := range content.Entries {
		if entry.Part == "" || parts[entry.Part] {
			return xlsxPresenceError(errors.New("content declarations are ambiguous"))
		}
		parts[entry.Part] = true
		if xlsxWorkbookContentType(entry.Type) {
			workbooks++
			if entry.Part != "/xl/workbook.xml" {
				return xlsxPresenceError(errors.New("workbook location is unsupported"))
			}
		}
	}
	if workbooks != 1 {
		return xlsxPresenceError(errors.New("workbook declarations are ambiguous"))
	}
	entry, err := selectedXLSXWorksheetEntry(archive, sheet)
	if err != nil {
		return err
	}
	var worksheet struct {
		XMLName xml.Name `xml:"worksheet"`
	}
	return decodeXLSXPart(archive, entry, &worksheet)
}

func xlsxWorkbookContentType(value string) bool {
	switch value {
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.template.main+xml",
		"application/vnd.ms-excel.sheet.macroEnabled.main+xml",
		"application/vnd.ms-excel.template.macroEnabled.main+xml":
		return true
	default:
		return false
	}
}
