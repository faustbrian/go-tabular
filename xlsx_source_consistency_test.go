package tabular

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func sourceRevisionWorkbook(t *testing.T, sheet, value string) ([]byte, int64) {
	t.Helper()
	base := makeErrorXLSX(t)
	index, err := zip.NewReader(bytes.NewReader(base), int64(len(base)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, entry := range index.File {
		input, openErr := entry.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		contents, readErr := io.ReadAll(input)
		closeErr := input.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("fixture member: %v, %v", readErr, closeErr)
		}
		if entry.Name == "xl/workbook.xml" {
			contents = bytes.ReplaceAll(contents, []byte(`name="Errors"`), []byte(`name="`+sheet+`"`))
		}
		if entry.Name == "xl/worksheets/sheet1.xml" {
			contents = bytes.ReplaceAll(contents, []byte("Value"), []byte(value))
		}
		destination, createErr := writer.CreateHeader(&zip.FileHeader{Name: entry.Name, Method: zip.Store})
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, err = destination.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	data := output.Bytes()
	index, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range index.File {
		if entry.Name == "xl/worksheets/sheet1.xml" {
			offset, offsetErr := entry.DataOffset()
			if offsetErr != nil {
				t.Fatal(offsetErr)
			}
			return data, offset - 30 - int64(len(entry.Name))
		}
	}
	t.Fatal("fixture worksheet missing")
	return nil, 0
}

type workbookRevisionSource struct {
	first, second []byte
	headerOffset  int64
	headerReads   int
	reads         int
	switched      bool
	closes        int
}

func (source *workbookRevisionSource) ReadAt(buffer []byte, offset int64) (int, error) {
	source.reads++
	// The prior constructor separately admitted worksheet syntax and graph,
	// then took Excelize's snapshot from offset zero of the borrowed source.
	if offset == 0 && source.headerReads == 2 {
		source.switched = true
	}
	if source.switched {
		return bytes.NewReader(source.second).ReadAt(buffer, offset)
	}
	if offset == source.headerOffset && len(buffer) == 30 {
		source.headerReads++
	}
	return bytes.NewReader(source.first).ReadAt(buffer, offset)
}

func (source *workbookRevisionSource) Close() error {
	source.closes++
	return nil
}

func TestXLSXAdmissionAndIterationUseOneSourceRevision(t *testing.T) {
	first, offset := sourceRevisionWorkbook(t, "Alpha", "Value")
	second, _ := sourceRevisionWorkbook(t, "Bravo", "Other")
	if len(first) != len(second) || len(first)+len(second) > 128*1024 || len(first) > 65537 {
		t.Fatal("ordinary revisions must be small and equal-sized")
	}
	for _, presence := range []bool{false, true} {
		source := &workbookRevisionSource{first: first, second: second, headerOffset: offset}
		reader, err := OpenSpreadsheet(source, int64(len(first)), SpreadsheetConfig{Format: FormatXLSX, Sheet: "Alpha", PreserveCellPresence: presence, MaxWorkbookBytes: int64(len(first)), ZIP: ZIPConfig{MaxArchiveBytes: int64(len(first))}})
		if err != nil {
			t.Fatalf("presence %v mixed admission and projection revisions: %v", presence, err)
		}
		t.Cleanup(func() {
			if closeErr := reader.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		})
		readsAtOpen := source.reads
		source.switched = true
		if presence {
			row, readErr := reader.ReadCells()
			if readErr != nil || len(row) != 1 || row[0].Value() != "Value" || !row[0].Present() {
				t.Fatalf("owned presence revision: %v, %v", row, readErr)
			}
		} else {
			row, readErr := reader.Read()
			if readErr != nil || len(row) != 1 || row[0] != "Value" {
				t.Fatalf("owned row revision: %v, %v", row, readErr)
			}
		}
		if err = reader.Close(); err != nil {
			t.Fatal(err)
		}
		if source.reads != readsAtOpen || source.closes != 0 {
			t.Fatalf("borrowed source touched after open: reads %d/%d closes %d", source.reads, readsAtOpen, source.closes)
		}
	}
}

type snapshotErrorSource struct {
	*bytes.Reader
	failure error
	reads   int
	closes  int
}

func (source *snapshotErrorSource) ReadAt(buffer []byte, offset int64) (int, error) {
	source.reads++
	if source.failure != nil {
		return 0, source.failure
	}
	return source.Reader.ReadAt(buffer, offset)
}

func (source *snapshotErrorSource) Close() error { source.closes++; return nil }

func TestXLSXSnapshotAdmissionAndReadFailure(t *testing.T) {
	data := makeErrorXLSX(t)
	for _, presence := range []bool{false, true} {
		for _, budget := range []string{"workbook", "archive"} {
			source := &snapshotErrorSource{Reader: bytes.NewReader(data)}
			config := SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence}
			if budget == "workbook" {
				config.MaxWorkbookBytes = int64(len(data) - 1)
			} else {
				config.ZIP.MaxArchiveBytes = int64(len(data) - 1)
			}
			reader, err := OpenSpreadsheet(source, int64(len(data)), config)
			if reader != nil || !errors.Is(err, ErrorLimitExceeded) || source.reads != 0 || source.closes != 0 {
				t.Fatalf("presence %v %s pre-read admission: %v", presence, budget, err)
			}
		}
		failure := errors.New("application-private-snapshot-read")
		for _, test := range []struct {
			failure error
			size    int64
			want    error
		}{{failure, int64(len(data)), failure}, {nil, int64(len(data) + 1), io.ErrUnexpectedEOF}} {
			source := &snapshotErrorSource{Reader: bytes.NewReader(data), failure: test.failure}
			reader, err := OpenSpreadsheet(source, test.size, SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: presence})
			if reader != nil {
				if closeErr := reader.Close(); closeErr != nil {
					t.Fatal(closeErr)
				}
			}
			if reader != nil || !errors.Is(err, ErrorArchive) || !errors.Is(err, test.want) || source.closes != 0 {
				t.Fatalf("presence %v snapshot failure: %v", presence, err)
			}
			if strings.Contains(err.Error(), failure.Error()) {
				t.Fatalf("private snapshot cause exposed: %v", err)
			}
		}
	}
}
