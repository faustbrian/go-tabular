package tabular

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

type bytewiseReader struct{ io.Reader }

func (reader bytewiseReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return reader.Reader.Read(p)
}

func TestCommentRecordAdmissionParity(t *testing.T) {
	for _, comment := range []rune{'#', 'Å'} {
		for _, split := range []bool{false, true} {
			data := string(comment) + ",\"\n" + string(comment) + "\n" + string(comment) + "\na\n"
			var source io.Reader = strings.NewReader(data)
			if split {
				source = bytewiseReader{source}
			}
			reader, err := NewCSVReader(source, DelimitedConfig{Comment: comment, MaxRecordBytes: 8})
			if err != nil {
				t.Fatal(err)
			}
			row, err := reader.Read()
			if err != nil || !reflect.DeepEqual(row, Row{"a"}) {
				t.Fatalf("comment %q split %v: %v %v", comment, split, row, err)
			}
		}
	}
}

func TestDelimitedSmallRecordFieldAndQuoteBudgets(t *testing.T) {
	for _, split := range []bool{false, true} {
		for _, test := range []struct {
			data   string
			config DelimitedConfig
			want   Row
			limit  bool
		}{
			{"\"a\nb\"\n", DelimitedConfig{Comment: '#', MaxRecordBytes: 6}, Row{"a\nb"}, false},
			{"\"a\nb\"\n", DelimitedConfig{Comment: '#', MaxRecordBytes: 5}, nil, true},
			{"ab\n", DelimitedConfig{MaxFieldBytes: 2}, Row{"ab"}, false},
			{"ab\n", DelimitedConfig{MaxFieldBytes: 1}, nil, true},
			{"\"#\n\"\n", DelimitedConfig{Comment: '#', MaxRecordBytes: 5}, Row{"#\n"}, false},
			{"\u2002\"a\nb\"\n", DelimitedConfig{Comment: '\u2003', TrimLeadingSpace: true, MaxRecordBytes: 9}, Row{"a\nb"}, false},
			{"\u2002\"a\nb\"\n", DelimitedConfig{Comment: '\u2003', TrimLeadingSpace: true, MaxRecordBytes: 8}, nil, true},
		} {
			var source io.Reader = strings.NewReader(test.data)
			if split {
				source = bytewiseReader{source}
			}
			reader, err := NewCSVReader(source, test.config)
			if err != nil {
				t.Fatal(err)
			}
			row, err := reader.Read()
			if test.limit {
				if row != nil || !errors.Is(err, ErrorLimitExceeded) {
					t.Fatalf("small refusal %q: %v %v", test.data, row, err)
				}
			} else if err != nil || !reflect.DeepEqual(row, test.want) {
				t.Fatalf("small acceptance %q: %v %v", test.data, row, err)
			}
		}
	}
}

func TestXLSXRelationshipAdmissionParity(t *testing.T) {
	data := transformZIP(t, makeErrorXLSX(t), func(name string, contents []byte) ([]byte, bool) {
		if name == "xl/_rels/workbook.xml.rels" {
			contents = []byte(strings.ReplaceAll(string(contents), `Target="worksheets/sheet1.xml"`, `Target="worksheets/sheet1.xml" TargetMode="External"`))
		}
		return contents, true
	})
	for _, presence := range []bool{false, true} {
		reader, err := OpenSpreadsheet(strings.NewReader(string(data)), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
		if reader != nil {
			_ = reader.Close()
		}
		if reader != nil || !errors.Is(err, ErrorSpreadsheet) {
			t.Fatalf("presence %v admitted external relationship: %v", presence, err)
		}
	}
}

func TestDelimitedAggregateAdmission(t *testing.T) {
	for _, header := range []bool{false, true} {
		for _, split := range []bool{false, true} {
			for _, test := range []struct {
				data    string
				maximum int64
				limit   bool
			}{
				{"\n#\na", 4, false}, {"\n#\nab", 4, true}, {"\n\n\n\n\na", 5, true},
			} {
				var input io.Reader = strings.NewReader(test.data)
				if split {
					input = bytewiseReader{input}
				}
				config := DelimitedConfig{Comment: '#', MaxSourceBytes: test.maximum}
				if header {
					config.Header = &HeaderConfig{}
				}
				reader, err := NewCSVReader(input, config)
				if err != nil {
					t.Fatal(err)
				}
				var row Row
				if header {
					row, err = reader.Header()
				} else {
					row, err = reader.Read()
				}
				if test.limit {
					if row != nil || !errors.Is(err, ErrorLimitExceeded) {
						t.Fatalf("%q header %v split %v: %v %v", test.data, header, split, row, err)
					}
					if header {
						row, err = reader.Header()
					} else {
						row, err = reader.Read()
					}
					if row != nil || !errors.Is(err, ErrorLimitExceeded) {
						t.Fatal("limit not sticky")
					}
				} else {
					if err != nil || !reflect.DeepEqual(row, Row{"a"}) {
						t.Fatalf("inclusive %q: %v %v", test.data, row, err)
					}
					_, err = reader.Read()
					if !errors.Is(err, io.EOF) {
						t.Fatalf("exact EOF: %v", err)
					}
				}
			}
		}
	}
	if _, err := NewCSVReader(strings.NewReader("a"), DelimitedConfig{MaxSourceBytes: -1}); !errors.Is(err, ErrorInvalidConfig) {
		t.Fatal("negative source limit accepted")
	}
}

func TestXLSMaterializationAdmissionPublic(t *testing.T) {
	data, err := os.ReadFile("testdata/spreadsheet/table.xls")
	if err != nil {
		t.Fatal(err)
	}
	for _, presence := range []bool{false, true} {
		reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLS, PreserveCellPresence: presence, MaxMaterializedCells: 1})
		if reader != nil || !errors.Is(err, ErrorLimitExceeded) {
			t.Fatalf("XLS budget presence %v: %v", presence, err)
		}
	}
}

func TestXLSXUnsafeTargetAdmissionParity(t *testing.T) {
	data := transformZIP(t, makeErrorXLSX(t), func(name string, contents []byte) ([]byte, bool) {
		if name == "xl/_rels/workbook.xml.rels" {
			contents = []byte(strings.ReplaceAll(string(contents), `Target="worksheets/sheet1.xml"`, `Target="worksheets/../worksheets/sheet1.xml"`))
		}
		return contents, true
	})
	for _, presence := range []bool{false, true} {
		reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
		if reader != nil {
			_ = reader.Close()
		}
		if reader != nil || !errors.Is(err, ErrorSpreadsheet) {
			t.Fatalf("unsafe target presence %v: %v", presence, err)
		}
	}
}
