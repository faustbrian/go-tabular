package xls

import (
	"encoding/binary"
	"errors"
	"testing"
)

func TestBIFFInclusiveDimensionsAndColumns(t *testing.T) {
	for _, test := range []struct {
		name                 string
		firstRow, lastRow    uint32
		firstColumn, lastCol uint16
	}{
		{"empty dimensions", 0, 0, 0, 0},
		{"format endpoints", 0, 65536, 0, 256},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := make([]byte, 14)
			binary.LittleEndian.PutUint32(payload, test.firstRow)
			binary.LittleEndian.PutUint32(payload[4:], test.lastRow)
			binary.LittleEndian.PutUint16(payload[8:], test.firstColumn)
			binary.LittleEndian.PutUint16(payload[10:], test.lastCol)
			for _, presence := range []bool{false, true} {
				book, err := parseBIFF8Limited(singleAdmissionWorkbook(biffRecord(0x0200, payload)), presence, 1, 1)
				if err != nil || book == nil || len(book.Sheets) != 1 || len(book.Sheets[0].Rows) != 0 {
					t.Fatalf("presence %v: dimensions must not materialize cells: %v %v", presence, book, err)
				}
			}
		})
	}
	for _, test := range []struct {
		name   string
		record []byte
		width  int
	}{
		{"empty final ROW range", biffRecord(0x0208, rowPayload(0, 255, 255)), 255},
		{"ROW end column", biffRecord(0x0208, rowPayload(0, 255, 256)), 256},
		{"NUMBER final column", biffRecord(0x0203, numberPayload(0, 255, 7)), 256},
		{"MULBLANK final column", biffRecord(0x00be, mulBlankPayload(0, 255, 1)), 256},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, presence := range []bool{false, true} {
				book, err := parseBIFF8Limited(singleAdmissionWorkbook(test.record), presence, 1, test.width)
				if err != nil || book == nil || len(book.Sheets) != 1 {
					t.Fatalf("presence %v: inclusive column: %v %v", presence, book, err)
				}
				rows := book.Sheets[0].Rows
				if test.name == "MULBLANK final column" && !presence {
					if len(rows) != 0 {
						t.Fatalf("ignored blank produced rows: %v", rows)
					}
				} else if len(rows) != 1 || len(rows[0]) != test.width {
					t.Fatalf("literal materialized width: %v", rows)
				}
				if test.name == "NUMBER final column" && rows[0][255].Value != "7" {
					t.Fatalf("final-column value: %v", rows[0][255])
				}
				if book, err = parseBIFF8Limited(singleAdmissionWorkbook(test.record), presence, 1, test.width-1); book != nil || !errors.Is(err, ErrLimit) {
					t.Fatalf("presence %v: one-less cell budget: %v %v", presence, book, err)
				}
			}
		})
	}
}

func TestBIFFIgnoredBlankAdmissionPrefixes(t *testing.T) {
	// The default projection deliberately ignores blank values; admission still
	// owns their position/range prefix and charges their slots in either mode.
	for _, test := range []struct {
		name   string
		record []byte
	}{
		{"BLANK position", biffRecord(0x0201, make([]byte, 4))},
		{"MULBLANK range", biffRecord(0x00be, make([]byte, 6))},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := singleAdmissionWorkbook(test.record)
			book, err := parseBIFF8Limited(data, false, 1, 1)
			if err != nil || book == nil || len(book.Sheets) != 1 || len(book.Sheets[0].Rows) != 0 {
				t.Fatalf("ignored blank: %v %v", book, err)
			}
			if book, err = parseBIFF8Limited(data, true, 1, 1); book != nil || err == nil || errors.Is(err, ErrLimit) {
				t.Fatalf("presence requires complete semantic record: %v %v", book, err)
			}
		})
	}
	payload := make([]byte, 6)
	binary.LittleEndian.PutUint16(payload[2:], 1)
	for _, presence := range []bool{false, true} {
		book, err := parseBIFF8Limited(singleAdmissionWorkbook(biffRecord(0x00be, payload)), presence, 1, 1)
		if book != nil || err == nil || err.Error() != "invalid BIFF8 cell range" {
			t.Fatalf("presence %v: inverted range must fail admission: %v %v", presence, book, err)
		}
	}
}

func TestBIFFSameSheetCumulativeSlots(t *testing.T) {
	records := append(biffRecord(0x0208, rowPayload(0, 0, 2)), biffRecord(0x0208, rowPayload(1, 0, 0))...)
	data := singleAdmissionWorkbook(records)
	for _, presence := range []bool{false, true} {
		book, err := parseBIFF8Limited(data, presence, 1, 3)
		if err != nil || book == nil || len(book.Sheets) != 1 || len(book.Sheets[0].Rows) != 2 || len(book.Sheets[0].Rows[0]) != 2 || len(book.Sheets[0].Rows[1]) != 0 {
			t.Fatalf("presence %v: literal rows at exact budget: %v %v", presence, book, err)
		}
		if book, err = parseBIFF8Limited(data, presence, 1, 2); book != nil || !errors.Is(err, ErrLimit) {
			t.Fatalf("presence %v: prior row slots must remain charged: %v %v", presence, book, err)
		}
	}
}

func TestBIFFPolicyAndSpanAdmissionOrder(t *testing.T) {
	for _, presence := range []bool{false, true} {
		for _, limits := range [][2]int{{-1, 1}, {1, -1}} {
			book, err := parseBIFF8Limited([]byte{0}, presence, limits[0], limits[1])
			if book != nil || !errors.Is(err, ErrLimit) {
				t.Fatalf("negative policy must precede BIFF parsing: %v %v", book, err)
			}
		}
		data := singleAdmissionWorkbook(biffRecord(0x0203, numberPayload(0, 0, 1)))
		binary.LittleEndian.PutUint32(data[12:], 0)
		book, err := parseBIFF8Limited(data, presence, 1, 1)
		if book != nil || err == nil || err.Error() != "invalid worksheet offset" {
			t.Fatalf("globals cannot be a worksheet: %v %v", book, err)
		}

		data = admissionWorkbook()
		first := binary.LittleEndian.Uint32(data[12:])
		second := binary.LittleEndian.Uint32(data[25:])
		binary.LittleEndian.PutUint32(data[12:], second)
		binary.LittleEndian.PutUint32(data[25:], first)
		book, err = parseBIFF8Limited(data, presence, 2, 3)
		if err != nil || book == nil || len(book.Sheets) != 2 || len(book.Sheets[0].Rows) != 1 || len(book.Sheets[0].Rows[0]) != 0 || len(book.Sheets[1].Rows[0]) != 2 {
			t.Fatalf("reverse adjacent sheets are disjoint: %v %v", book, err)
		}
		// A duplicate must be rejected before charging its dense cells again.
		data = admissionWorkbook()
		binary.LittleEndian.PutUint32(data[25:], binary.LittleEndian.Uint32(data[12:]))
		book, err = parseBIFF8Limited(data, presence, 2, 2)
		if book != nil || err == nil || errors.Is(err, ErrLimit) || err.Error() != "overlapping worksheet offsets" {
			t.Fatalf("duplicate span admission must precede repeated charge: %v %v", book, err)
		}
	}
}

func TestBIFFWorksheetEOFOffsetCause(t *testing.T) {
	for _, test := range []struct {
		name  string
		extra int
		cause string
	}{
		{"at EOF", 0, "worksheet BOF not found"},
		{"past EOF", 1, "invalid worksheet offset"},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := singleAdmissionWorkbook(biffRecord(0x0203, numberPayload(0, 0, 1)))
			binary.LittleEndian.PutUint32(data[12:], uint32(len(data)+test.extra))
			for _, presence := range []bool{false, true} {
				book, err := parseBIFF8Limited(data, presence, 1, 1)
				if book != nil || err == nil || errors.Is(err, ErrLimit) || err.Error() != test.cause {
					t.Fatalf("presence %v: original worksheet admission cause: %v %v", presence, book, err)
				}
			}
		})
	}
}

func TestDIFATExactPhysicalSectorCount(t *testing.T) {
	// Two physical sectors are a complete two-link DIFAT chain. No workbook
	// projection is involved: this is the owned physical-chain admission seam.
	compound := &compoundFile{data: make([]byte, 3*512), sectorSize: 512}
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, freeSector)
	for _, id := range []uint32{0, 1} {
		sector, err := compound.sector(id)
		if err != nil {
			t.Fatal(err)
		}
		for offset := 0; offset < 512; offset += 4 {
			binary.LittleEndian.PutUint32(sector[offset:], freeSector)
		}
		if id == 0 {
			binary.LittleEndian.PutUint32(sector[508:], 1)
		} else {
			binary.LittleEndian.PutUint32(sector[508:], endOfChain)
		}
	}
	result, err := readDIFAT(compound, header, 0, 2, 0)
	if err != nil || result == nil || len(result) != 0 {
		t.Fatalf("complete physical chain: %v %v", result, err)
	}
	if result, err = readDIFAT(compound, header, 0, 3, 0); result != nil || err == nil || err.Error() != "xls: invalid DIFAT sector count" {
		t.Fatalf("one-over physical chain: %v %v", result, err)
	}
}
