package tabular

import (
	"archive/zip"
	"bytes"
	"errors"
	"hash/crc32"
	"io"
	"strings"
	"sync"
	"testing"
)

const presenceTestCompression uint16 = 65001

// archive/zip has no unregister operation. This method belongs only to this
// sequential fixture; registration never captures a test or scenario owner.
var presenceCodecRegistry struct {
	sync.Mutex
	once   sync.Once
	active *presenceCodecScenario
}

type presenceCodecScenario struct {
	sync.Mutex
	opens        int
	failAt       int
	cause        error
	failedCloses int
}

type presenceCodecReader struct {
	io.Reader
	owner  *presenceCodecScenario
	failed bool
}

func (reader *presenceCodecReader) Read(data []byte) (int, error) {
	if reader.failed {
		return 0, reader.owner.cause
	}
	return reader.Reader.Read(data)
}

func (reader *presenceCodecReader) Close() error {
	if reader.failed {
		reader.owner.Lock()
		reader.owner.failedCloses++
		reader.owner.Unlock()
	}
	return nil
}

func TestXLSXPresenceDecompressorFailureIsPrivate(t *testing.T) {
	presenceCodecRegistry.once.Do(func() {
		zip.RegisterDecompressor(presenceTestCompression, func(input io.Reader) io.ReadCloser {
			presenceCodecRegistry.Lock()
			owner := presenceCodecRegistry.active
			presenceCodecRegistry.Unlock()
			if owner == nil {
				return io.NopCloser(input)
			}
			owner.Lock()
			owner.opens++
			failed := owner.opens == owner.failAt
			owner.Unlock()
			return &presenceCodecReader{Reader: input, owner: owner, failed: failed}
		})
	})
	data := presenceCodecWorkbook(t)
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "presence refusal"}[fail], func(t *testing.T) {
			owner := &presenceCodecScenario{cause: errors.New("application-private-codec-detail")}
			if fail {
				// This named member is opened by sheet-count admission, graph
				// admission, SDK expansion, then presence admission after Rows.
				owner.failAt = 4
			}
			presenceCodecRegistry.Lock()
			presenceCodecRegistry.active = owner
			presenceCodecRegistry.Unlock()
			defer func() {
				presenceCodecRegistry.Lock()
				presenceCodecRegistry.active = nil
				presenceCodecRegistry.Unlock()
			}()
			reader, err := OpenSpreadsheet(bytes.NewReader(data), int64(len(data)), SpreadsheetConfig{Format: FormatXLSX, PreserveCellPresence: true})
			if reader != nil {
				defer func() {
					if closeErr := reader.Close(); closeErr != nil {
						t.Error(closeErr)
					}
				}()
			}
			if fail {
				var detail *Error
				if reader != nil || !errors.Is(err, ErrorSpreadsheet) || !errors.Is(err, owner.cause) || !errors.As(err, &detail) || detail.Op != "spreadsheet.presence" {
					t.Fatalf("presence refusal = %v, %v", reader, err)
				}
				if err.Error() != "tabular: spreadsheet.presence xlsx: spreadsheet error" || strings.Contains(err.Error(), owner.cause.Error()) {
					t.Fatalf("unsafe default error: %v", err)
				}
			} else {
				if err != nil || reader == nil {
					t.Fatalf("successful codec construction: %v", err)
				}
				row, readErr := reader.Read()
				if readErr != nil || len(row) != 1 || row[0] != "Value" {
					t.Fatalf("successful codec row = %v, %v", row, readErr)
				}
			}
			owner.Lock()
			opens, closed := owner.opens, owner.failedCloses
			owner.Unlock()
			if opens != 4 || (fail && closed != 1) || (!fail && closed != 0) {
				t.Fatalf("named workbook opens = %d; failed reader closes = %d", opens, closed)
			}
		})
	}
}

func presenceCodecWorkbook(t *testing.T) []byte {
	t.Helper()
	base := makeErrorXLSX(t)
	archive, err := zip.NewReader(bytes.NewReader(base), int64(len(base)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range archive.File {
		reader, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		contents, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("fixture member read: %v; close: %v", readErr, closeErr)
		}
		header := zip.FileHeader{Name: file.Name, Method: zip.Store, CRC32: crc32.ChecksumIEEE(contents), CompressedSize64: uint64(len(contents)), UncompressedSize64: uint64(len(contents))}
		if file.Name == "xl/workbook.xml" {
			header.Method = presenceTestCompression
		}
		entry, createErr := writer.CreateRaw(&header)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, err = entry.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if output.Len() > 65537 || len(base)+output.Len() > 128*1024 {
		t.Fatal("ordinary fixture exceeds test budget")
	}
	return output.Bytes()
}
