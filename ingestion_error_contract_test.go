package tabular

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestZIPChecksumPrivateDiagnosticCause(t *testing.T) {
	const name = "private-entry-marker"
	const payload = "private-payload-marker"
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.WriteString(entry, payload); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	data := buffer.Bytes()
	index, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	offset, err := index.File[0].DataOffset()
	if err != nil {
		t.Fatal(err)
	}
	data[offset] ^= 1
	archive, err := OpenZIP(bytes.NewReader(data), int64(len(data)), ZIPConfig{MaxArchiveBytes: 1024, MaxEntryBytes: 64, MaxTotalBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	reader, err := archive.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)
	if closeErr := reader.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	want := []byte(payload)
	want[0] ^= 1
	if !bytes.Equal(got, want) {
		t.Fatalf("partial bytes %q", got)
	}
	if !errors.Is(err, ErrorArchive) || !errors.Is(err, zip.ErrChecksum) || errors.Unwrap(err) != zip.ErrChecksum {
		t.Fatalf("checksum cause lost: %v", err)
	}
	var detail *Error
	if !errors.As(err, &detail) || err.Error() != "tabular: zip.entry.read zip: archive error" || strings.Contains(err.Error(), name) || strings.Contains(err.Error(), payload) {
		t.Fatalf("default diagnostic %v", err)
	}
}

func TestCSVPartialMalformedFieldAdmission(t *testing.T) {
	reader, err := NewCSVReader(strings.NewReader(`aa,"`), DelimitedConfig{MaxFieldBytes: 1, MaxRecordBytes: 64, MaxSourceBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	row, err := reader.Read()
	var detail *Error
	if row != nil || !errors.Is(err, ErrorLimitExceeded) || !errors.As(err, &detail) || detail.Row != 1 || detail.Field != 1 || err.Error() != "tabular: delimited.read csv row 1 field 1: limit exceeded" {
		t.Fatalf("partial field rejection: %v %v", row, err)
	}
}

func TestCSVQuotedRunAdmission(t *testing.T) {
	for _, bytewise := range []bool{false, true} {
		for _, maximum := range []int{6, 3} {
			var source io.Reader = strings.NewReader("\"aaa\"\n")
			if bytewise {
				source = bytewiseReader{source}
			}
			reader, err := NewCSVReader(source, DelimitedConfig{MaxRecordBytes: maximum, MaxFieldBytes: 8, MaxSourceBytes: 64})
			if err != nil {
				t.Fatal(err)
			}
			row, err := reader.Read()
			if maximum == 6 {
				if err != nil || !reflect.DeepEqual(row, Row{"aaa"}) {
					t.Fatalf("inclusive quoted record: %v %v", row, err)
				}
			} else if row != nil || !errors.Is(err, ErrorLimitExceeded) {
				t.Fatalf("quoted record limit: %v %v", row, err)
			}
		}
	}
}

func TestCSVClosingQuoteGrammarParity(t *testing.T) {
	for _, test := range []struct {
		name, text string
		comma      rune
	}{
		{"CRLF", "\"a\"\r\n", ','},
		{"CR non-LF", "\"a\"\rX\n", ','},
		{"wrong delimiter", "\"a\"x\n", ','},
		{"Unicode continuation mismatch", "\"a\"\u2002x\n", '\u2003'},
		{"Unicode delimiter", "\"a\"\u2003b\n", '\u2003'},
	} {
		t.Run(test.name, func(t *testing.T) {
			standard := csv.NewReader(strings.NewReader(test.text))
			standard.Comma = test.comma
			want, wantErr := standard.Read()
			for _, bytewise := range []bool{false, true} {
				var source io.Reader = strings.NewReader(test.text)
				if bytewise {
					source = bytewiseReader{source}
				}
				reader, err := NewDelimitedReader(source, DelimitedConfig{Delimiter: test.comma, MaxRecordBytes: 64, MaxFieldBytes: 16, MaxSourceBytes: 64})
				if err != nil {
					t.Fatal(err)
				}
				got, err := reader.Read()
				if wantErr == nil {
					if err != nil || !reflect.DeepEqual(got, Row(want)) {
						t.Fatalf("grammar parity: %v %v", got, err)
					}
					if _, err = reader.Read(); !errors.Is(err, io.EOF) {
						t.Fatalf("EOF: %v", err)
					}
					continue
				}
				var actual, expected *csv.ParseError
				if got != nil || !errors.Is(err, ErrorMalformedRow) || !errors.As(err, &actual) || !errors.As(wantErr, &expected) || actual.StartLine != expected.StartLine || actual.Line != expected.Line || actual.Column != expected.Column || actual.Err != expected.Err || err.Error() != "tabular: delimited.read delimited row 1: malformed row" {
					t.Fatalf("malformed grammar parity: %v %v; standard %v", got, err, wantErr)
				}
			}
		})
	}
}
