package xls

import (
	"strconv"
	"testing"
)

func TestOLECountAdmissionPreservesHostCapacity(t *testing.T) {
	for _, test := range []struct {
		name      string
		count     uint32
		available int
		want      bool
	}{
		{name: "empty", count: 0, available: 0, want: true},
		{name: "exact", count: 1, available: 1, want: true},
		{name: "one over", count: 2, available: 1, want: false},
		{name: "negative capacity", count: 0, available: -1, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := countFits(test.count, test.available); got != test.want {
				t.Fatalf("count admission = %v, want %v", got, test.want)
			}
		})
	}
	t.Run("capacity above wire width", func(t *testing.T) {
		if strconv.IntSize < 64 {
			t.Skip("this scalar capacity is representable only on 64-bit hosts")
		}
		capacity := uint64(1) << 32
		if !countFits(1, int(capacity)) {
			t.Fatal("positive count refused despite available host capacity")
		}
	})
}
