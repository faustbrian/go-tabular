package xls

import (
	"encoding/binary"
	"errors"
)

// ErrLimit identifies admission refusal before worksheet materialization.
var ErrLimit = errors.New("xls: materialization limit exceeded")

const defaultMaxCells = 1_000_000
const defaultMaxSheets = 128

// admitSheet checks BIFF8 dimensions and charges dense cell slots, including a
// minimum slot for each absent/empty row. It never builds cell values or dense
// output. remaining is workbook-wide, including unselected worksheets.
func admitSheet(data []byte, offset int, remaining *int) (int, error) {
	first, err := readRecord(data, offset)
	if err != nil || first.id != 0x0809 {
		return 0, errors.New("worksheet BOF not found")
	}
	widths := make(map[int]int)
	maxRow := -1
	slots := 0
	for {
		rec, err := readRecord(data, offset)
		if err != nil {
			return 0, err
		}
		if rec.id == 0x000a {
			*remaining -= slots
			return rec.next, nil
		}
		row, width := -1, 0
		switch rec.id {
		case 0x0200:
			if len(rec.payload) != 14 {
				return 0, errors.New("invalid DIMENSIONS record")
			}
			firstRow := binary.LittleEndian.Uint32(rec.payload[:4])
			lastRow := binary.LittleEndian.Uint32(rec.payload[4:8])
			firstCol := binary.LittleEndian.Uint16(rec.payload[8:10])
			lastCol := binary.LittleEndian.Uint16(rec.payload[10:12])
			if firstRow > lastRow || lastRow > 65536 || firstCol > lastCol || lastCol > 256 {
				return 0, errors.New("invalid BIFF8 dimensions")
			}
		case 0x0208:
			if len(rec.payload) < 6 {
				return 0, errors.New("truncated ROW record")
			}
			row = int(binary.LittleEndian.Uint16(rec.payload[:2]))
			first := int(binary.LittleEndian.Uint16(rec.payload[2:4]))
			width = int(binary.LittleEndian.Uint16(rec.payload[4:6]))
			if first > 255 || width > 256 || first > width {
				return 0, errors.New("invalid BIFF8 ROW columns")
			}
		case 0x00fd, 0x0203, 0x027e, 0x0201, 0x0205:
			if len(rec.payload) < 4 {
				return 0, errors.New("truncated cell position")
			}
			row = int(binary.LittleEndian.Uint16(rec.payload[:2]))
			width = int(binary.LittleEndian.Uint16(rec.payload[2:4])) + 1
			if width > 256 {
				return 0, errors.New("invalid BIFF8 cell column")
			}
		case 0x00bd, 0x00be:
			if len(rec.payload) < 6 {
				return 0, errors.New("truncated cell range")
			}
			row = int(binary.LittleEndian.Uint16(rec.payload[:2]))
			first := int(binary.LittleEndian.Uint16(rec.payload[2:4]))
			width = int(binary.LittleEndian.Uint16(rec.payload[len(rec.payload)-2:])) + 1
			if first > 255 || width > 256 || first >= width {
				return 0, errors.New("invalid BIFF8 cell range")
			}
		}
		if row >= 0 {
			if row > maxRow {
				slots += row - maxRow
				maxRow = row
			}
			old := max(1, widths[row])
			next := max(1, width)
			if next > old {
				slots += next - old
			}
			if slots > *remaining {
				return 0, ErrLimit
			}
			widths[row] = max(widths[row], width)
		}
		offset = rec.next
	}
}
