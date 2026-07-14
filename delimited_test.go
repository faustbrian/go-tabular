package tabular

import (
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestDelimitedReaderStreamsFixtureWithNormalizedHeader(t *testing.T) {
	t.Parallel()

	file, err := os.Open("testdata/delimited/realistic.csv")
	if err != nil {
		t.Fatal(err)
	}
	closeTestResource(t, file)

	reader, err := NewCSVReader(file, DelimitedConfig{
		Header: &HeaderConfig{
			TrimSpace:        true,
			Case:             HeaderCaseLower,
			RejectEmpty:      true,
			RejectDuplicates: true,
		},
		Normalize: NormalizationConfig{TrimSpace: true},
	})
	if err != nil {
		t.Fatalf("NewCSVReader() error = %v", err)
	}

	header, err := reader.Header()
	if err != nil {
		t.Fatalf("Header() error = %v", err)
	}
	if want := (Row{"name", "city", "note"}); !reflect.DeepEqual(header, want) {
		t.Fatalf("Header() = %#v, want %#v", header, want)
	}

	rows, err := readAllRows(reader)
	if err != nil {
		t.Fatalf("read rows: %v", err)
	}
	want := []Row{
		{"Alice", "Helsinki", "Uses, commas"},
		{"Björk", "Reykjavík", "quoted\nnewline"},
		{"Matti", "Espoo", ""},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}

	header[0] = "changed"
	headerAgain, err := reader.Header()
	if err != nil {
		t.Fatalf("second Header() error = %v", err)
	}
	if headerAgain[0] != "name" {
		t.Fatal("Header() returned mutable internal state")
	}
}

func TestDelimitedReaderSupportsSemicolonAndComments(t *testing.T) {
	t.Parallel()

	file, err := os.Open("testdata/delimited/semicolon.csv")
	if err != nil {
		t.Fatal(err)
	}
	closeTestResource(t, file)

	reader, err := NewDelimitedReader(file, DelimitedConfig{
		Delimiter:           ';',
		Comment:             '#',
		TrimLeadingSpace:    true,
		AllowVariableFields: true,
	})
	if err != nil {
		t.Fatalf("NewDelimitedReader() error = %v", err)
	}

	rows, err := readAllRows(reader)
	if err != nil {
		t.Fatalf("read rows: %v", err)
	}
	want := []Row{{"id", "amount", "description"}, {"1", "12,50", "Nordic order"}, {"2", "", "trailing", ""}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func TestDelimitedReaderReportsMalformedAndWrongShapeRows(t *testing.T) {
	t.Parallel()
	malformed, err := os.ReadFile("testdata/delimited/malformed.csv")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		input string
		row   int
	}{
		{name: "wrong shape", input: "a,b\n1\n", row: 2},
		{name: "bad quote fixture", input: string(malformed), row: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			reader, err := NewCSVReader(strings.NewReader(test.input), DelimitedConfig{})
			if err != nil {
				t.Fatalf("NewCSVReader() error = %v", err)
			}
			if _, err = reader.Read(); err != nil {
				t.Fatalf("first Read() error = %v", err)
			}
			_, err = reader.Read()
			if !errors.Is(err, ErrorMalformedRow) {
				t.Fatalf("Read() error = %v, want malformed-row kind", err)
			}
			var tabularErr *Error
			if !errors.As(err, &tabularErr) || tabularErr.Row != test.row {
				t.Fatalf("Read() error = %#v, want row %d", err, test.row)
			}
		})
	}
}

func TestDelimitedReaderHandlesEmptyInputsDeterministically(t *testing.T) {
	t.Parallel()

	withoutHeader, err := NewCSVReader(strings.NewReader(""), DelimitedConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = withoutHeader.Read(); !errors.Is(err, io.EOF) {
		t.Fatalf("Read() error = %v, want EOF", err)
	}

	withHeader, err := NewCSVReader(strings.NewReader(""), DelimitedConfig{Header: &HeaderConfig{}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = withHeader.Header()
	if !errors.Is(err, ErrorInvalidHeader) || !errors.Is(err, io.EOF) {
		t.Fatalf("Header() error = %v, want invalid-header wrapping EOF", err)
	}
	if _, readErr := withHeader.Read(); !errors.Is(readErr, ErrorInvalidHeader) {
		t.Fatalf("Read() error = %v, want cached header error %v", readErr, err)
	}
}

func TestDelimitedReaderValidatesConfiguration(t *testing.T) {
	t.Parallel()

	tests := []DelimitedConfig{
		{},
		{Delimiter: '"'},
		{Delimiter: ',', Comment: ','},
		{Delimiter: ',', Comment: '\r'},
	}
	for _, config := range tests {
		_, err := NewDelimitedReader(strings.NewReader(""), config)
		if !errors.Is(err, ErrorInvalidConfig) {
			t.Fatalf("NewDelimitedReader(%+v) error = %v, want invalid config", config, err)
		}
	}
}

func TestDelimitedReaderCanUseLazyQuotesExplicitly(t *testing.T) {
	t.Parallel()

	reader, err := NewCSVReader(strings.NewReader("a,b\n1,unquoted\"quote\n"), DelimitedConfig{
		LazyQuotes: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := readAllRows(reader)
	if err != nil {
		t.Fatal(err)
	}
	if want := []Row{{"a", "b"}, {"1", "unquoted\"quote"}}; !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

func FuzzDelimitedReader(f *testing.F) {
	f.Add(",", "a,b\n1,2\n")
	f.Add(";", "name;city\nBjörk;Reykjavík\n")
	f.Fuzz(func(t *testing.T, delimiterText, input string) {
		delimiter := []rune(delimiterText)
		if len(delimiter) != 1 {
			t.Skip()
		}
		reader, err := NewDelimitedReader(strings.NewReader(input), DelimitedConfig{
			Delimiter:           delimiter[0],
			AllowVariableFields: true,
		})
		if err != nil {
			return
		}
		for {
			_, err = reader.Read()
			if err != nil {
				return
			}
		}
	})
}

func BenchmarkDelimitedReader(b *testing.B) {
	data := strings.Repeat("1,Alice,Helsinki\n", 20_000)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for range b.N {
		reader, err := NewCSVReader(strings.NewReader(data), DelimitedConfig{})
		if err != nil {
			b.Fatal(err)
		}
		if err = consumeRows(reader); err != nil {
			b.Fatal(err)
		}
	}
}

func consumeRows(reader interface{ Read() (Row, error) }) error {
	for {
		_, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func readAllRows(reader interface{ Read() (Row, error) }) ([]Row, error) {
	var rows []Row
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
}
