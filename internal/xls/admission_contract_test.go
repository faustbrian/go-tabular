package xls

import (
	"encoding/binary"
	"errors"
	"testing"
)

func singleAdmissionWorkbook(record []byte) []byte {
	bof := biffRecord(0x0809, []byte{0, 6, 5, 0})
	eof := biffRecord(0x000a, nil)
	definition := biffRecord(0x0085, append(make([]byte, 6), 1, 0, 'A'))
	binary.LittleEndian.PutUint32(definition[4:], uint32(len(bof)+len(definition)+len(eof)))
	data := append(bof, definition...)
	data = append(data, eof...)
	data = append(data, biffRecord(0x0809, nil)...)
	data = append(data, record...)
	return append(data, eof...)
}

func TestBIFFDimensionsAndSemanticPayloadRefusal(t *testing.T) {
	for _, test := range []struct {
		name           string
		record         []byte
		cause, message string
	}{
		{"short dimensions", biffRecord(0x0200, make([]byte, 13)), "", "invalid DIMENSIONS record"},
		{"admitted short LABELSST", biffRecord(0x00fd, make([]byte, 9)), "invalid LABELSST record", `xls: sheet "A": invalid LABELSST record`},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, presence := range []bool{false, true} {
				book, err := parseBIFF8Limited(singleAdmissionWorkbook(test.record), presence, 1, 1)
				if book != nil || err == nil || errors.Is(err, ErrLimit) || err.Error() != test.message {
					t.Fatalf("invalid grammar: %v %v", book, err)
				}
				if test.cause != "" && (errors.Unwrap(err) == nil || errors.Unwrap(err).Error() != test.cause) {
					t.Fatalf("semantic cause: %v", err)
				}
			}
		})
	}
}

func TestBIFFNegativePolicyClassification(t *testing.T) {
	data := singleAdmissionWorkbook(biffRecord(0x0203, numberPayload(0, 0, 1)))
	for _, presence := range []bool{false, true} {
		book, err := parseBIFF8Limited(data, presence, 1, 1)
		if err != nil || book == nil || len(book.Sheets) != 1 || len(book.Sheets[0].Rows) != 1 || len(book.Sheets[0].Rows[0]) != 1 {
			t.Fatalf("positive policy: %v %v", book, err)
		}
		for _, limits := range [][2]int{{-1, 1}, {1, -1}} {
			book, err = parseBIFF8Limited(data, presence, limits[0], limits[1])
			if book != nil || !errors.Is(err, ErrLimit) {
				t.Fatalf("negative policy: %v %v", book, err)
			}
		}
	}
}

func TestBIFFReverseOrderedOverlapRefusal(t *testing.T) {
	bof := biffRecord(0x0809, []byte{0, 6, 5, 0})
	eof := biffRecord(0x000a, nil)
	inner := biffRecord(0x0085, append(make([]byte, 6), 1, 0, 'A'))
	outer := biffRecord(0x0085, append(make([]byte, 6), 1, 0, 'B'))
	start := len(bof) + len(inner) + len(outer) + len(eof)
	sheetBOF := biffRecord(0x0809, nil)
	binary.LittleEndian.PutUint32(inner[4:], uint32(start+len(sheetBOF)))
	binary.LittleEndian.PutUint32(outer[4:], uint32(start))
	data := append(bof, inner...)
	data = append(data, outer...)
	data = append(data, eof...)
	data = append(data, sheetBOF...)
	data = append(data, sheetBOF...)
	data = append(data, eof...)
	data = append(data, eof...)
	for _, presence := range []bool{false, true} {
		book, err := parseBIFF8Limited(data, presence, 2, 1)
		if book != nil || err == nil || err.Error() != "overlapping worksheet offsets" {
			t.Fatalf("reverse overlap: %v %v", book, err)
		}
	}
}
