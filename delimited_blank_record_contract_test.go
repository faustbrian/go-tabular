package tabular

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestCSVBlankLineResetsRecordByteBudget(t *testing.T) {
	for _, input := range []string{"a\n", "\na\n"} {
		reader, err := NewCSVReader(strings.NewReader(input), DelimitedConfig{MaxRecordBytes: 2})
		if err != nil {
			t.Fatal(err)
		}
		row, err := reader.Read()
		if err != nil || len(row) != 1 || row[0] != "a" {
			t.Fatalf("input %q first row = %v, %v", input, row, err)
		}
		row, err = reader.Read()
		if row != nil || !errors.Is(err, io.EOF) {
			t.Fatalf("input %q terminal row = %v, %v", input, row, err)
		}
	}
}
