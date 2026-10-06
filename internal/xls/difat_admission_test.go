package xls

import (
	"encoding/binary"
	"testing"
)

func TestDIFATPhysicalAdmission(t *testing.T) {
	compound := &compoundFile{data: make([]byte, 3*512), sectorSize: 512}
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, freeSector)
	for _, test := range []struct {
		count, fat uint32
		want       string
	}{
		{3, 0, "xls: invalid DIFAT sector count"}, {0, 3, "xls: invalid FAT sector count"},
	} {
		t.Run(test.want, func(t *testing.T) {
			result, err := readDIFAT(compound, header, endOfChain, test.count, test.fat)
			if result != nil || err == nil || err.Error() != test.want {
				t.Fatalf("count %d FAT %d: %v %v", test.count, test.fat, result, err)
			}
		})
	}
}
