package tabular

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestXLSXCompleteGraphDocumentContracts(t *testing.T) {
	for _, test := range []struct {
		name, part, old, replacement string
		valid                        bool
	}{
		{"directive", "xl/worksheets/sheet1.xml", "", "<!DOCTYPE worksheet>", false},
		{"trailing text", "xl/worksheets/sheet1.xml", "", "unexpected", false},
		{"trailing comment", "xl/worksheets/sheet1.xml", "", "<!-- complete worksheet -->", true},
		{"missing office document", "_rels/.rels", "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument", "http://schemas.openxmlformats.org/officeDocument/2006/relationships/metadata", false},
		{"malformed content types", "[Content_Types].xml", "", "<Types><Override", false},
		{"missing workbook type", "[Content_Types].xml", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml", "application/xml", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := transformZIP(t, makeErrorXLSX(t), func(name string, contents []byte) ([]byte, bool) {
				if name == test.part {
					switch test.name {
					case "directive":
						contents = append([]byte(test.replacement), contents...)
					case "trailing text", "trailing comment":
						contents = append(contents, []byte(test.replacement)...)
					case "malformed content types":
						contents = []byte(test.replacement)
					default:
						contents = []byte(strings.Replace(string(contents), test.old, test.replacement, 1))
					}
				}
				return contents, true
			})
			for _, presence := range []bool{false, true} {
				reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence, PreserveCellErrors: true, MaxWorkbookBytes: 4096, MaxSheets: 2, ZIP: ZIPConfig{MaxArchiveBytes: 4096, MaxEntryBytes: 4096, MaxTotalBytes: 8192}})
				if !test.valid {
					if reader != nil {
						_ = reader.Close()
					}
					var detail *Error
					if reader != nil || !errors.Is(err, ErrorSpreadsheet) || !errors.As(err, &detail) || errors.Unwrap(err) == nil || err.Error() != "tabular: "+detail.Op+" xlsx: spreadsheet error" || len(err.Error()) > 96 {
						t.Fatalf("complete graph refusal: %v", err)
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				for _, want := range []Row{{"Value"}, {"#DIV/0!"}} {
					got, err := reader.Read()
					if err != nil || !reflect.DeepEqual(got, want) {
						t.Fatalf("trailing comment parity: %v %v", got, err)
					}
				}
				if _, err = reader.Read(); !errors.Is(err, io.EOF) {
					t.Fatalf("trailing comment EOF: %v", err)
				}
				if err = reader.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestXLSXGraphAmbiguityRefusal(t *testing.T) {
	for _, test := range []struct{ name, part, old, replacement string }{
		{"different root workbook", "_rels/.rels", `Target="xl/workbook.xml"`, `Target="xl/other.xml"`},
		{"duplicate workbook declaration", "[Content_Types].xml", "</Types>", `<Override PartName="/xl/other.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/></Types>`},
		{"duplicate content part", "[Content_Types].xml", "</Types>", `<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/></Types>`},
		{"duplicate root ID", "_rels/.rels", "</Relationships>", `<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"duplicate sheet name", "xl/workbook.xml", "</sheets>", `<sheet name="Errors" sheetId="2" r:id="rId2"/></sheets>`},
		{"duplicate sheet relationship", "xl/workbook.xml", "</sheets>", `<sheet name="Other" sheetId="2" r:id="rId1"/></sheets>`},
		{"duplicate sheet ID", "xl/workbook.xml", "</sheets>", `<sheet name="Other" sheetId="1" r:id="rId2"/></sheets>`},
		{"duplicate relationship ID", "xl/_rels/workbook.xml.rels", "</Relationships>", `<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := transformZIP(t, makeErrorXLSX(t), func(name string, contents []byte) ([]byte, bool) {
				if name == test.part {
					contents = []byte(strings.Replace(string(contents), test.old, test.replacement, 1))
				}
				return contents, true
			})
			for _, presence := range []bool{false, true} {
				reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, Sheet: "Errors", PreserveCellPresence: presence})
				if reader != nil {
					_ = reader.Close()
				}
				if reader != nil || !errors.Is(err, ErrorSpreadsheet) {
					t.Fatalf("presence %v graph ambiguity admitted: %v", presence, err)
				}
			}
		})
	}
}

func TestXLSXIndependentGraphAdmissionPredicates(t *testing.T) {
	base := makeErrorXLSX(t)
	totalBytes := len(base)
	for _, test := range []struct {
		name, part, old, replacement, cause string
	}{
		{name: "canonical control"},
		{"blank root ID", "_rels/.rels", `Id="rId1"`, `Id=""`, "root relationships are ambiguous"},
		{"external workbook", "_rels/.rels", `Target="xl/workbook.xml"`, `Target="xl/workbook.xml" TargetMode="External"`, "workbook location is unsupported"},
		{"unsupported office document type", "_rels/.rels", `http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument`, `urn:application-private/officeDocument`, "workbook location is unsupported"},
		{"blank non-workbook content part", "[Content_Types].xml", `PartName="/xl/worksheets/sheet1.xml"`, `PartName=""`, "content declarations are ambiguous"},
		{"translated root relationship ID", "_rels/.rels", `Id="rId1"`, `Id="http://purl.oclc.org/ooxml/drawingml/main"`, "relationship identity is unsupported"},
		{"translated unselected worksheet relationship ID", "xl/_rels/workbook.xml.rels", "</Relationships>", `<Relationship Id="http://purl.oclc.org/ooxml/officeDocument/docPropsVTypes" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="custom/unused.xml"/></Relationships>`, "relationship identity is unsupported"},
		{"translated worksheet target", "xl/_rels/workbook.xml.rels", `Target="worksheets/sheet1.xml"`, `Target="custom/http://purl.oclc.org/ooxml/officeDocument/extendedProperties.xml"`, "relationship identity is unsupported"},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := base
			if test.part != "" {
				data = transformZIP(t, base, func(name string, contents []byte) ([]byte, bool) {
					if name == test.part {
						if !strings.Contains(string(contents), test.old) {
							t.Fatal("fixture does not contain selected graph attribute")
						}
						contents = []byte(strings.Replace(string(contents), test.old, test.replacement, 1))
					}
					return contents, true
				})
				totalBytes += len(data)
			}
			if len(data) > 65537 || totalBytes > 128*1024 {
				t.Fatal("ordinary graph fixtures exceed test budget")
			}
			for _, presence := range []bool{false, true} {
				reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
				if reader != nil {
					defer func() {
						if closeErr := reader.Close(); closeErr != nil {
							t.Error(closeErr)
						}
					}()
				}
				if test.cause == "" {
					if err != nil || reader == nil {
						t.Fatalf("presence %v canonical construction: %v", presence, err)
					}
					row, readErr := reader.Read()
					if readErr != nil || len(row) != 1 || row[0] != "Value" {
						t.Fatalf("presence %v canonical row = %v, %v", presence, row, readErr)
					}
					continue
				}
				var detail *Error
				if reader != nil || !errors.Is(err, ErrorSpreadsheet) || !errors.As(err, &detail) || detail.Op != "spreadsheet.presence" {
					t.Fatalf("presence %v canonical graph refusal = %v, %v", presence, reader, err)
				}
				cause := errors.Unwrap(err)
				if cause == nil || cause.Error() != test.cause || !errors.Is(detail.Err, cause) {
					t.Fatalf("presence %v canonical admission cause = %v", presence, cause)
				}
				if err.Error() != "tabular: spreadsheet.presence xlsx: spreadsheet error" || strings.Contains(err.Error(), "application-private") {
					t.Fatalf("presence %v unsafe default diagnostic: %v", presence, err)
				}
			}
		})
	}
}

func TestXLSXRelationshipIdentityTranslationFamilies(t *testing.T) {
	for _, strict := range []string{
		"http://purl.oclc.org/ooxml/officeDocument/docPropsVTypes",
		"http://purl.oclc.org/ooxml/drawingml/main",
		"http://purl.oclc.org/ooxml/officeDocument/extendedProperties",
		"http://purl.oclc.org/ooxml/spreadsheetml/main",
		"http://purl.oclc.org/ooxml/officeDocument/relationships",
		"http://purl.oclc.org/ooxml/officeDocument/relationships/chart",
		"http://purl.oclc.org/ooxml/officeDocument/relationships/comments",
		"http://purl.oclc.org/ooxml/officeDocument/relationships/extendedProperties",
		"http://purl.oclc.org/ooxml/officeDocument/relationships/image",
		"http://purl.oclc.org/ooxml/officeDocument/relationships/officeDocument",
	} {
		if stableXLSXRelationshipIdentity("prefix/" + strict + "/suffix") {
			t.Fatalf("translated identity accepted: %q", strict)
		}
	}
	for _, unchanged := range []string{"rId1", "worksheets/sheet1.xml", "http://schemas.openxmlformats.org/spreadsheetml/2006/main", "http://purl.oclc.org/custom"} {
		if !stableXLSXRelationshipIdentity(unchanged) {
			t.Fatalf("unchanged identity refused: %q", unchanged)
		}
	}
}

func TestXLSXTranslatedRelationshipIdentityRefused(t *testing.T) {
	const strictID = "http://purl.oclc.org/ooxml/spreadsheetml/main"
	const delegateID = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	const worksheetType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet"
	const other = `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Delegate</t></is></c></row></sheetData></worksheet>`
	base := renameZIPEntry(t, makeErrorXLSX(t), "xl/worksheets/sheet1.xml", "xl/custom/admitted.xml")
	data := transformZIP(t, base, func(name string, contents []byte) ([]byte, bool) {
		switch name {
		case "xl/workbook.xml":
			contents = []byte(strings.Replace(string(contents), `r:id="rId1"`, `r:id="`+strictID+`"`, 1))
		case "xl/_rels/workbook.xml.rels":
			contents = []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="` + strictID + `" Type="` + worksheetType + `" Target="custom/admitted.xml"/><Relationship Id="` + delegateID + `" Type="` + worksheetType + `" Target="custom/other.xml"/></Relationships>`)
		case "[Content_Types].xml":
			contents = []byte(strings.Replace(string(contents), "worksheets/sheet1.xml", "custom/admitted.xml", 1))
		case "xl/custom/admitted.xml":
			contents = []byte(strings.Replace(string(contents), "Value", "Admission", 1))
		}
		return contents, true
	})
	data = addZIPEntry(t, data, "xl/custom/other.xml", other)
	if len(data) > 65537 || len(base)+len(data) > 128*1024 {
		t.Fatal("ordinary relationship fixtures exceed test budget")
	}
	for _, presence := range []bool{false, true} {
		t.Run(fmt.Sprint(presence), func(t *testing.T) {
			reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
			if reader != nil {
				row, readErr := reader.Read()
				if closeErr := reader.Close(); closeErr != nil {
					t.Error(closeErr)
				}
				t.Fatalf("translated relationship admitted row %v, read error %v; expected nil constructor", row, readErr)
			}
			var detail *Error
			if !errors.Is(err, ErrorSpreadsheet) || !errors.As(err, &detail) || detail.Op != "spreadsheet.presence" {
				t.Fatalf("relationship identity refusal: %v", err)
			}
			cause := errors.Unwrap(err)
			if cause == nil || cause.Error() != "relationship identity is unsupported" || err.Error() != "tabular: spreadsheet.presence xlsx: spreadsheet error" {
				t.Fatalf("relationship refusal category/cause: %v, %v", err, cause)
			}
		})
	}
}

func TestXLSXStrictNamespaceVocabularyRemainsSupported(t *testing.T) {
	data := transformZIP(t, makeErrorXLSX(t), func(_ string, contents []byte) ([]byte, bool) {
		text := strings.ReplaceAll(string(contents), "http://schemas.openxmlformats.org/spreadsheetml/2006/main", "http://purl.oclc.org/ooxml/spreadsheetml/main")
		text = strings.ReplaceAll(text, "http://schemas.openxmlformats.org/officeDocument/2006/relationships", "http://purl.oclc.org/ooxml/officeDocument/relationships")
		return []byte(text), true
	})
	if len(data) > 65537 {
		t.Fatal("ordinary strict workbook exceeds test budget")
	}
	for _, presence := range []bool{false, true} {
		reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
		if err != nil || reader == nil {
			t.Fatalf("presence %v strict vocabulary construction: %v", presence, err)
		}
		row, readErr := reader.Read()
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || len(row) != 1 || row[0] != "Value" {
			t.Fatalf("presence %v strict vocabulary row = %v; read %v; close %v", presence, row, readErr, closeErr)
		}
	}
}

func TestXLSXUnrelatedIDNamespaceCannotSelectAnotherWorksheet(t *testing.T) {
	const worksheetType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet"
	const other = `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Delegate</t></is></c></row></sheetData></worksheet>`
	base := renameZIPEntry(t, makeErrorXLSX(t), "xl/worksheets/sheet1.xml", "xl/custom/admitted.xml")
	data := transformZIP(t, base, func(name string, contents []byte) ([]byte, bool) {
		switch name {
		case "xl/workbook.xml":
			contents = []byte(strings.Replace(string(contents), `r:id="rId1"`, `xmlns:x="urn:application" r:id="rId2" x:id="rId1"`, 1))
		case "xl/_rels/workbook.xml.rels":
			contents = []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="` + worksheetType + `" Target="custom/admitted.xml"/><Relationship Id="rId2" Type="` + worksheetType + `" Target="custom/other.xml"/></Relationships>`)
		case "[Content_Types].xml":
			contents = []byte(strings.Replace(string(contents), "worksheets/sheet1.xml", "custom/admitted.xml", 1))
		case "xl/custom/admitted.xml":
			contents = []byte(strings.Replace(string(contents), "Value", "Admission", 1))
		}
		return contents, true
	})
	data = addZIPEntry(t, data, "xl/custom/other.xml", other)
	if len(data) > 65537 || len(base)+len(data) > 128*1024 {
		t.Fatal("ordinary namespace fixtures exceed test budget")
	}
	for _, presence := range []bool{false, true} {
		t.Run(fmt.Sprint(presence), func(t *testing.T) {
			reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
			if reader != nil {
				row, readErr := reader.Read()
				if closeErr := reader.Close(); closeErr != nil {
					t.Error(closeErr)
				}
				t.Fatalf("unrelated ID namespace admitted row %v, read error %v; expected nil constructor", row, readErr)
			}
			var detail *Error
			if !errors.Is(err, ErrorSpreadsheet) || !errors.As(err, &detail) || detail.Op != "spreadsheet.presence" || errors.Unwrap(err) == nil || err.Error() != "tabular: spreadsheet.presence xlsx: spreadsheet error" {
				t.Fatalf("private namespace admission refusal: %v", err)
			}
		})
	}
}

func TestXLSXWorksheetRelationshipAttributeNamespaces(t *testing.T) {
	base := makeErrorXLSX(t)
	totalBytes := len(base)
	for _, test := range []struct{ name, attributes, cause string }{
		{"transitional", `r:id="rId1"`, ""},
		{"strict", `xmlns:s="http://purl.oclc.org/ooxml/officeDocument/relationships" s:id="rId1"`, ""},
		{"namespace prefix named id", `xmlns:id="urn:application" r:id="rId1"`, ""},
		{"unrelated before recognized", `xmlns:x="urn:application" x:id="rId2" r:id="rId1"`, "worksheet relationship namespace is invalid"},
		{"missing", "", "worksheet declaration is invalid"},
		{"unqualified", `id="rId1"`, "worksheet relationship namespace is invalid"},
		{"dual recognized different", `xmlns:s="http://purl.oclc.org/ooxml/officeDocument/relationships" r:id="rId1" s:id="rId2"`, "worksheet relationship identities are ambiguous"},
		{"dual recognized equal", `xmlns:s="http://purl.oclc.org/ooxml/officeDocument/relationships" s:id="rId1" r:id="rId1"`, "worksheet relationship identities are ambiguous"},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := transformZIP(t, base, func(name string, contents []byte) ([]byte, bool) {
				if name == "xl/workbook.xml" {
					contents = []byte(strings.Replace(string(contents), `r:id="rId1"`, test.attributes, 1))
				}
				return contents, true
			})
			totalBytes += len(data)
			if len(data) > 65537 || totalBytes > 128*1024 {
				t.Fatal("ordinary attribute fixtures exceed test budget")
			}
			for _, presence := range []bool{false, true} {
				reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
				if test.cause == "" {
					if err != nil || reader == nil {
						t.Fatalf("presence %v valid namespace: %v", presence, err)
					}
					row, readErr := reader.Read()
					closeErr := reader.Close()
					if readErr != nil || closeErr != nil || len(row) != 1 || row[0] != "Value" {
						t.Fatalf("presence %v namespace row = %v; read %v; close %v", presence, row, readErr, closeErr)
					}
					continue
				}
				if reader != nil {
					if closeErr := reader.Close(); closeErr != nil {
						t.Error(closeErr)
					}
				}
				var detail *Error
				if reader != nil || !errors.Is(err, ErrorSpreadsheet) || !errors.As(err, &detail) || detail.Op != "spreadsheet.presence" {
					t.Fatalf("presence %v namespace refusal: %v", presence, err)
				}
				cause := errors.Unwrap(err)
				if cause == nil || cause.Error() != test.cause || err.Error() != "tabular: spreadsheet.presence xlsx: spreadsheet error" {
					t.Fatalf("presence %v namespace refusal cause: %v, %v", presence, err, cause)
				}
			}
		})
	}
}

func relocatedWorksheet(t *testing.T, malformed bool) []byte {
	return transformZIP(t, renameZIPEntry(t, makeErrorXLSX(t), "xl/worksheets/sheet1.xml", "xl/custom/sheet1.xml"), func(name string, contents []byte) ([]byte, bool) {
		if name == "xl/_rels/workbook.xml.rels" {
			contents = []byte(strings.ReplaceAll(string(contents), "worksheets/sheet1.xml", "custom/sheet1.xml"))
		}
		if name == "[Content_Types].xml" {
			contents = []byte(strings.ReplaceAll(string(contents), "worksheets/sheet1.xml", "custom/sheet1.xml"))
		}
		if name == "xl/custom/sheet1.xml" && malformed {
			contents = []byte(`<worksheet><sheetData>`)
		}
		return contents, true
	})
}

func TestXLSXSelectedPartCompleteValidation(t *testing.T) {
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"relocated malformed", relocatedWorksheet(t, true)},
		{"trailing malformed", transformZIP(t, makeErrorXLSX(t), func(name string, contents []byte) ([]byte, bool) {
			if name == "xl/worksheets/sheet1.xml" {
				contents = append(contents, []byte("</")...)
			}
			return contents, true
		})},
		{"second document", transformZIP(t, makeErrorXLSX(t), func(name string, contents []byte) ([]byte, bool) {
			if name == "xl/worksheets/sheet1.xml" {
				contents = append(contents, []byte("<worksheet/>")...)
			}
			return contents, true
		})},
		{"leading non-XML content", transformZIP(t, makeErrorXLSX(t), func(name string, contents []byte) ([]byte, bool) {
			if name == "xl/worksheets/sheet1.xml" {
				contents = append([]byte("unexpected"), contents...)
			}
			return contents, true
		})},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, presence := range []bool{false, true} {
				reader, err := OpenSpreadsheet(bytes.NewReader(test.data), int64(len(test.data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
				if reader != nil {
					_ = reader.Close()
				}
				if reader != nil || !errors.Is(err, ErrorSpreadsheet) {
					t.Fatalf("presence %v malformed selected XML admitted: %v", presence, err)
				}
			}
		})
	}
}

func TestXLSXValidRelocatedWorksheetParity(t *testing.T) {
	data := relocatedWorksheet(t, false)
	for _, presence := range []bool{false, true} {
		reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence, PreserveCellErrors: true})
		if err != nil {
			t.Fatal(err)
		}
		row, err := reader.Read()
		if err != nil || !reflect.DeepEqual(row, Row{"Value"}) {
			t.Fatalf("first relocated row %v %v", row, err)
		}
		row, err = reader.Read()
		if err != nil || !reflect.DeepEqual(row, Row{"#DIV/0!"}) {
			t.Fatalf("second relocated row %v %v", row, err)
		}
		if _, err = reader.Read(); !errors.Is(err, io.EOF) {
			t.Fatalf("relocated EOF %v", err)
		}
		if err = reader.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func relativeXLWorksheet(t *testing.T, malformed bool) []byte {
	data := transformZIP(t, relocatedWorksheet(t, false), func(name string, contents []byte) ([]byte, bool) {
		if name == "xl/_rels/workbook.xml.rels" {
			contents = []byte(strings.ReplaceAll(string(contents), `Target="custom/sheet1.xml"`, `Target="xl/custom/sheet1.xml"`))
		}
		if name == "[Content_Types].xml" {
			contents = []byte(strings.ReplaceAll(string(contents), `/xl/custom/sheet1.xml`, `/xl/xl/custom/sheet1.xml`))
		}
		return contents, true
	})
	part := `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Actual</t></is></c></row></sheetData></worksheet>`
	if malformed {
		part = `<worksheet><sheetData>`
	}
	return addZIPEntry(t, data, "xl/xl/custom/sheet1.xml", part)
}

func TestXLSXRelativeXLTargetParity(t *testing.T) {
	data := relativeXLWorksheet(t, false)
	for _, presence := range []bool{false, true} {
		t.Run(fmt.Sprint(presence), func(t *testing.T) {
			reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = reader.Close() }()
			row, err := reader.Read()
			if err != nil || !reflect.DeepEqual(row, Row{"Actual"}) {
				t.Fatalf("actual selected row %v %v", row, err)
			}
			if _, err = reader.Read(); !errors.Is(err, io.EOF) {
				t.Fatalf("actual selected EOF %v", err)
			}
		})
	}
}

func TestXLSXRelativeXLTargetMalformedRefusal(t *testing.T) {
	data := relativeXLWorksheet(t, true)
	for _, presence := range []bool{false, true} {
		t.Run(fmt.Sprint(presence), func(t *testing.T) {
			reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
			if reader != nil {
				_ = reader.Close()
			}
			if reader != nil || !errors.Is(err, ErrorSpreadsheet) {
				t.Fatalf("malformed actual part admitted: %v", err)
			}
		})
	}
}

func TestXLSXUnicodeSheetNameAmbiguity(t *testing.T) {
	data := transformZIP(t, makeErrorXLSX(t), func(name string, contents []byte) ([]byte, bool) {
		if name == "xl/workbook.xml" {
			text := strings.ReplaceAll(string(contents), `name="Errors"`, `name="S"`)
			contents = []byte(strings.Replace(text, "</sheets>", `<sheet name="ſ" sheetId="2" r:id="rId2"/></sheets>`, 1))
		}
		if name == "xl/_rels/workbook.xml.rels" {
			contents = []byte(strings.Replace(string(contents), "</Relationships>", `<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`, 1))
		}
		return contents, true
	})
	for _, presence := range []bool{false, true} {
		t.Run(fmt.Sprint(presence), func(t *testing.T) {
			reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, Sheet: "S", PreserveCellPresence: presence})
			if reader != nil {
				_ = reader.Close()
			}
			if reader != nil || !errors.Is(err, ErrorSpreadsheet) {
				t.Fatalf("Unicode-equivalent names admitted: %v", err)
			}
		})
	}
}
