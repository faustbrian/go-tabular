package xls

import "testing"

func TestOLEChainRejectsUnrepresentableSectorID(t *testing.T) {
	compound, err := parseCompoundFile(xlsFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if compound.sectorSize != 512 {
		t.Fatal("ordinary fixture must use 512-byte sectors")
	}
	chain, err := compound.chain(0x80000000, compound.fat, compound.sectorCount())
	if chain != nil || err == nil || err.Error() != "xls: invalid sector chain" {
		t.Fatalf("out-of-range wire ID admission = %v, %v", chain, err)
	}
}

func TestOLESectorRejectsUnrepresentableSectorID(t *testing.T) {
	compound, err := parseCompoundFile(xlsFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if compound.sectorSize != 512 {
		t.Fatal("ordinary fixture must use 512-byte sectors")
	}
	sector, err := compound.sector(0x80000000)
	if sector != nil || err == nil || err.Error() != "xls: sector outside file" {
		t.Fatalf("out-of-range wire ID admission = %v, %v", sector, err)
	}
}
