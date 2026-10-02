package tabular

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
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

func TestXLSXRowsRefusesUnsupportedSheetNamePrivately(t *testing.T) {
	data := rewriteZIPEntry(t, makeErrorXLSX(t), "xl/workbook.xml", `<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="A/B" sheetId="1" r:id="rId1"/></sheets></workbook>`)
	if len(data) > 65537 {
		t.Fatal("ordinary workbook fixture exceeds its test budget")
	}
	archive, err := OpenZIP(bytes.NewReader(data), int64(len(data)), ZIPConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if err = validateXLSXWorksheets(archive); err != nil {
		t.Fatal(err)
	}
	if err = validateXLSXGraph(archive, ""); err != nil {
		t.Fatal(err)
	}
	for _, presence := range []bool{false, true} {
		reader, openErr := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
		if reader != nil {
			if closeErr := reader.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
		}
		if reader != nil || !errors.Is(openErr, ErrorSpreadsheet) {
			t.Fatalf("presence %v rows refusal: %v", presence, openErr)
		}
		if openErr.Error() != "tabular: spreadsheet.rows xlsx: spreadsheet error" || strings.Contains(openErr.Error(), "A/B") {
			t.Fatalf("presence %v default diagnostic: %v", presence, openErr)
		}
		cause := errors.Unwrap(openErr)
		if !errors.Is(openErr, excelize.ErrSheetNameInvalid) || !errors.Is(cause, excelize.ErrSheetNameInvalid) {
			t.Fatalf("presence %v rows cause missing: %v", presence, cause)
		}
	}
}

// Excelize translates strict namespace URI text throughout workbook XML,
// including this malformed sheet-name value. Admission sees the original name;
// public construction must refuse the changed projection without echoing it.
func TestXLSXNamespaceSheetNameProjectionRefusedPrivately(t *testing.T) {
	const name = "http://purl.oclc.org/ooxml/spreadsheetml/main"
	base := makeErrorXLSX(t)
	data := rewriteZIPEntry(t, base, "xl/workbook.xml", `<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="`+name+`" sheetId="1" r:id="rId1"/></sheets></workbook>`)
	if len(base)+len(data) > 128*1024 || len(data) > 65537 {
		t.Fatal("ordinary workbook fixture exceeds its test budget")
	}
	for _, presence := range []bool{false, true} {
		reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, Sheet: name, PreserveCellPresence: presence})
		if reader != nil {
			if closeErr := reader.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
		}
		var detail *Error
		if reader != nil || !errors.Is(err, ErrorSpreadsheet) || !errors.As(err, &detail) || detail.Op != "spreadsheet.presence" {
			t.Fatalf("presence %v same-byte projection refusal: %v", presence, err)
		}
		if err.Error() != "tabular: spreadsheet.presence xlsx: spreadsheet error" || strings.Contains(err.Error(), name) {
			t.Fatalf("presence %v private default diagnostic: %v", presence, err)
		}
		cause := errors.Unwrap(err)
		if cause == nil || cause.Error() != "relationship identity is unsupported" || !errors.Is(detail.Err, cause) {
			t.Fatalf("presence %v trusted refusal cause missing: %v", presence, cause)
		}
	}
}

// Refuse the old borrowed-archive reopening boundary after its two worksheet
// admissions. Snapshot-based construction must never reach that caller read.
type worksheetReopenFailure struct {
	*bytes.Reader
	headerOffset int64
	headerReads  int
	failure      error
}

func (r *worksheetReopenFailure) ReadAt(p []byte, offset int64) (int, error) {
	if offset == r.headerOffset && len(p) == 30 {
		r.headerReads++
		if r.headerReads > 2 {
			return 0, r.failure
		}
	}
	return r.Reader.ReadAt(p, offset)
}

func TestXLSXSnapshotDoesNotReopenBorrowedWorksheet(t *testing.T) {
	data := makeErrorXLSX(t)
	index, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var headerOffset int64 = -1
	for _, entry := range index.File {
		if entry.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		dataOffset, offsetErr := entry.DataOffset()
		if offsetErr != nil {
			t.Fatal(offsetErr)
		}
		headerOffset = dataOffset - 30 - int64(len(entry.Name))
		if headerOffset < 0 || binary.LittleEndian.Uint32(data[headerOffset:]) != 0x04034b50 || binary.LittleEndian.Uint16(data[headerOffset+28:]) != 0 {
			t.Fatal("fixture must have a normal worksheet header without extra fields")
		}
	}
	if headerOffset < 0 || len(data) > 65537 {
		t.Fatal("ordinary worksheet fixture missing or too large")
	}
	failure := errors.New("application-private-source-read")
	for _, presence := range []bool{false, true} {
		source := &worksheetReopenFailure{Reader: bytes.NewReader(data), headerOffset: headerOffset, failure: failure}
		reader, openErr := OpenSpreadsheet(source, int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
		if openErr != nil || reader == nil {
			t.Fatalf("presence %v owned snapshot: %v", presence, openErr)
		}
		row, readErr := reader.Read()
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || len(row) != 1 || row[0] != "Value" || source.headerReads != 0 {
			t.Fatalf("presence %v snapshot read: %v, %v, %v; borrowed header reads %d", presence, row, readErr, closeErr, source.headerReads)
		}
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
