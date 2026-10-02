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
