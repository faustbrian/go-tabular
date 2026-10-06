package xls

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

func TestBIFFRowDimensionAdmission(t *testing.T) {
	payload := make([]byte, 6)
	binary.LittleEndian.PutUint16(payload[4:], 257)
	data := append(biffRecord(0x0809, nil), biffRecord(0x0208, payload)...)
	data = append(data, biffRecord(0x000a, nil)...)
	for _, presence := range []bool{false, true} {
		rows, positions, err := parseSheetState(data, 0, nil, presence)
		if err == nil || rows != nil || positions != nil {
			t.Fatalf("presence %v accepted invalid BIFF8 ROW width", presence)
		}
	}
}

func TestOpenCompatibilityAndBudgetCause(t *testing.T) {
	data := xlsFixture(t)
	for _, presence := range []bool{false, true} {
		var old *Workbook
		var err error
		if presence {
			old, err = OpenWithPresence(data)
		} else {
			old, err = Open(data)
		}
		if err != nil || old == nil || len(old.Sheets) == 0 {
			t.Fatalf("retained parser: %v", err)
		}
		bounded, err := OpenBounded(data, presence, 0, 0)
		if err != nil || !reflect.DeepEqual(old, bounded) {
			t.Fatalf("default parser parity: %v", err)
		}
		if result, err := OpenBounded(data, presence, 0, 1); result != nil || !errors.Is(err, ErrLimit) {
			t.Fatalf("budget cause: %v", err)
		}
		if result, err := open([]byte("not xls"), presence); result != nil || err == nil {
			t.Fatal("malformed compound accepted")
		}
	}
}

func admissionWorkbook() []byte {
	bof := biffRecord(0x0809, []byte{0, 6, 5, 0})
	eof := biffRecord(0x000a, nil)
	left := biffRecord(0x0085, append(make([]byte, 6), 1, 0, 'A'))
	right := biffRecord(0x0085, append(make([]byte, 6), 1, 0, 'B'))
	first := append(biffRecord(0x0809, nil), biffRecord(0x0208, rowPayload(0, 0, 2))...)
	first = append(first, eof...)
	second := append(biffRecord(0x0809, nil), biffRecord(0x0208, rowPayload(0, 0, 0))...)
	second = append(second, eof...)
	start := len(bof) + len(left) + len(right) + len(eof)
	binary.LittleEndian.PutUint32(left[4:], uint32(start))
	binary.LittleEndian.PutUint32(right[4:], uint32(start+len(first)))
	data := append(bof, left...)
	data = append(data, right...)
	data = append(data, eof...)
	data = append(data, first...)
	return append(data, second...)
}

func TestBIFFCumulativeAdmission(t *testing.T) {
	for _, presence := range []bool{false, true} {
		data := admissionWorkbook()
		book, err := parseBIFF8Limited(data, presence, 2, 3)
		if err != nil || len(book.Sheets) != 2 || len(book.Sheets[0].Rows[0]) != 2 || len(book.Sheets[1].Rows) != 1 {
			t.Fatalf("inclusive: %v", err)
		}
		for _, limits := range [][2]int{{2, 2}, {1, 3}} {
			book, err = parseBIFF8Limited(data, presence, limits[0], limits[1])
			if book != nil || !errors.Is(err, ErrLimit) {
				t.Fatalf("cumulative refused: %v", err)
			}
		}
		// Both BOUNDSHEET records referring to the same BOF must not repeat work.
		binary.LittleEndian.PutUint32(data[25:], binary.LittleEndian.Uint32(data[12:]))
		book, err = parseBIFF8Limited(data, presence, 2, 10)
		if book != nil || err == nil {
			t.Fatal("duplicate worksheet offset accepted")
		}
	}
}

func TestBIFFCoordinateAdmission(t *testing.T) {
	for _, rec := range [][]byte{
		biffRecord(0x00fd, labelSSTPayload(0, 256, 0)), biffRecord(0x0203, numberPayload(0, 256, 1)),
		biffRecord(0x0201, blankPayload(0, 256)), biffRecord(0x00be, mulBlankPayload(0, 255, 2)),
		biffRecord(0x0208, rowPayload(0, 2, 1)),
	} {
		data := append(biffRecord(0x0809, nil), rec...)
		data = append(data, biffRecord(0x000a, nil)...)
		for _, presence := range []bool{false, true} {
			rows, _, err := parseSheetState(data, 0, []string{"a"}, presence)
			if rows != nil || err == nil {
				t.Fatal("out-of-format columns admitted")
			}
		}
	}
	data := append(biffRecord(0x0809, nil), biffRecord(0x0208, rowPayload(2, 0, 0))...)
	data = append(data, biffRecord(0x000a, nil)...)
	remaining := 2
	if _, err := admitSheet(data, 0, &remaining); !errors.Is(err, ErrLimit) {
		t.Fatal("absent row slots not charged")
	}
	for _, invalid := range []bool{false, true} {
		payload := make([]byte, 14)
		binary.LittleEndian.PutUint32(payload[4:], 3)
		binary.LittleEndian.PutUint16(payload[10:], 2)
		if invalid {
			binary.LittleEndian.PutUint32(payload[4:], 65537)
		}
		data := append(biffRecord(0x0809, nil), biffRecord(0x0200, payload)...)
		data = append(data, biffRecord(0x000a, nil)...)
		for _, presence := range []bool{false, true} {
			rows, _, err := parseSheetState(data, 0, nil, presence)
			if invalid {
				if rows != nil || err == nil {
					t.Fatal("out-of-format DIMENSIONS admitted")
				}
			} else if err != nil {
				t.Fatalf("valid DIMENSIONS: %v", err)
			}
		}
	}
}
