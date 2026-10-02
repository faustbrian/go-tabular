package tabular

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestDelimitedSourceEmptyReadDoesNotConsumeInput(t *testing.T) {
	source := &countingReader{Reader: strings.NewReader("a")}
	reader := &delimitedSourceLimitReader{source: source, remaining: 1}
	for _, buffer := range [][]byte{nil, {}} {
		count, err := reader.Read(buffer)
		if count != 0 || err != nil || source.reads != 0 || reader.remaining != 1 {
			t.Fatalf("empty read = %d, %v; source reads %d, remaining %d", count, err, source.reads, reader.remaining)
		}
	}
	buffer := make([]byte, 1)
	count, err := reader.Read(buffer)
	if count != 1 || err != nil || string(buffer) != "a" || source.reads != 1 || reader.remaining != 0 {
		t.Fatalf("following read = %d, %v, %q; source reads %d, remaining %d", count, err, buffer, source.reads, reader.remaining)
	}
}

func TestXLSXStylesDelegateFailurePreservesPrivateCause(t *testing.T) {
	const marker = "application-private-style-count"
	base := makeErrorXLSX(t)
	index, err := zip.NewReader(bytes.NewReader(base), int64(len(base)))
	if err != nil {
		t.Fatal(err)
	}
	files := make(map[string]string)
	for _, entry := range index.File {
		reader, openErr := entry.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		contents, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("base member: %v, %v", readErr, closeErr)
		}
		files[entry.Name] = string(contents)
	}
	files["xl/styles.xml"] = `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="` + marker + `"/></styleSheet>`
	data := makeZIP(t, files)
	if len(base)+len(data) > 128*1024 || len(data) > 65537 {
		t.Fatal("ordinary archive fixture exceeds its test budget")
	}
	archive, err := OpenZIP(bytes.NewReader(data), int64(len(data)), ZIPConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if err = validateXLSXSheetLimit(archive, 2); err != nil {
		t.Fatalf("sheet preflight: %v", err)
	}
	if err = validateXLSXWorksheets(archive); err != nil {
		t.Fatalf("worksheet preflight: %v", err)
	}
	if err = validateXLSXGraph(archive, ""); err != nil {
		t.Fatalf("graph preflight: %v", err)
	}
	for _, presence := range []bool{false, true} {
		reader, openErr := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
		if reader != nil {
			if closeErr := reader.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
		}
		if reader != nil || !errors.Is(openErr, ErrorSpreadsheet) {
			t.Fatalf("presence %v delegate refusal: %v", presence, openErr)
		}
		if openErr.Error() != "tabular: spreadsheet.open xlsx: spreadsheet error" || strings.Contains(openErr.Error(), marker) {
			t.Fatalf("presence %v default diagnostic: %v", presence, openErr)
		}
		cause := errors.Unwrap(openErr)
		if cause == nil || !strings.Contains(cause.Error(), marker) {
			t.Fatalf("presence %v trusted delegate cause missing", presence)
		}
	}
}
