package tabular

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

// The Postal application parses Posti fixed-width and Geposit semicolon
// sources with hand-written readers. These references reproduce exactly the
// framing those readers apply, so each new option is proven against the
// behaviour it replaces on the Postal parity fixtures and mutations of them.

var postiFields = []FixedWidthField{
	{Name: "postal_code", Start: 13, End: 18, TrimCutset: postiCutset},
	{Name: "name_fi", Start: 18, End: 48, TrimCutset: postiCutset},
	{Name: "name_sv", Start: 48, End: 78, TrimCutset: postiCutset},
	{Name: "state_fi", Start: 116, End: 146, TrimCutset: postiCutset},
	{Name: "state_sv", Start: 146, End: 176, TrimCutset: postiCutset},
}

const postiCutset = " \t\n\r\x00\v"

// postiReference mirrors Postal's posti.Decode framing: ScanLines records of
// exactly 176 or 220 bytes, each field trimmed of the cutset bytes and then
// read as Latin-1.
func postiReference(source []byte) ([]Row, error) {
	scanner := bufio.NewScanner(bytes.NewReader(source))
	scanner.Buffer(make([]byte, 176), 223)
	var rows []Row
	for scanner.Scan() {
		record := scanner.Bytes()
		if len(record) != 176 && len(record) != 220 {
			return rows, errors.New("malformed width")
		}
		row := make(Row, len(postiFields))
		for index, field := range postiFields {
			value := bytes.Trim(record[field.Start:field.End], postiCutset)
			var text strings.Builder
			for _, character := range value {
				text.WriteRune(rune(character))
			}
			row[index] = text.String()
		}
		rows = append(rows, row)
	}
	return rows, scanner.Err()
}

func postiLibrary(source []byte) ([]Row, error) {
	reader, err := NewFixedWidthReader(bytes.NewReader(source), FixedWidthConfig{
		Fields: postiFields, Encoding: EncodingISO88591, RecordWidths: []int{176, 220},
		MaxRecordBytes: 223,
	})
	if err != nil {
		return nil, err
	}
	var rows []Row
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return rows, err
		}
		rows = append(rows, row)
	}
}

// gepositReference mirrors Postal's geposit.Decode framing: every physical
// line is exactly one semicolon record, so blank lines and quoted fields that
// span lines are malformed.
func gepositReference(source []byte) ([]Row, error) {
	reader := bufio.NewReader(bytes.NewReader(source))
	var rows []Row
	for {
		line, err := reader.ReadSlice('\n')
		if len(line) == 0 && errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return rows, err
		}
		line = bytes.TrimSuffix(bytes.TrimSuffix(line, []byte{'\n'}), []byte{'\r'})
		if !utf8.Valid(line) {
			return rows, errors.New("invalid UTF-8")
		}
		parser := csv.NewReader(bytes.NewReader(line))
		parser.Comma = ';'
		parser.FieldsPerRecord = -1
		values, parseErr := parser.Read()
		if parseErr != nil {
			return rows, parseErr
		}
		if _, parseErr = parser.Read(); !errors.Is(parseErr, io.EOF) {
			return rows, errors.New("line holds more than one record")
		}
		rows = append(rows, Row(values))
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
	}
}

func gepositLibrary(source []byte) ([]Row, error) {
	reader, err := NewDelimitedReader(bytes.NewReader(source), DelimitedConfig{
		Delimiter: ';', AllowVariableFields: true, SingleLineRecords: true, RejectBlankLines: true,
	})
	if err != nil {
		return nil, err
	}
	var rows []Row
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return rows, err
		}
		rows = append(rows, row)
	}
}

func readParityFixture(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/postal-parity/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestFixedWidthRecordWidthsAndCutsetMatchPostalPosti(t *testing.T) {
	t.Parallel()

	fixture := readParityFixture(t, "posti.dat")
	lines := strings.SplitAfter(string(fixture), "\n")
	first := strings.TrimSuffix(lines[0], "\n")
	certified := first + strings.Repeat(" ", 44)
	padded := first[:13] + "\x00\v" + first[15:]
	for name, source := range map[string]string{
		"fixture":              string(fixture),
		"certified width":      certified + "\n" + string(fixture),
		"crlf":                 strings.ReplaceAll(string(fixture), "\n", "\r\n"),
		"nul and vertical tab": padded + "\n",
		"width 175":            first[:175] + "\n",
		"width 177":            first + " \n",
		"width 219":            certified[:219] + "\n",
		"width 221":            certified + " \n",
		"blank line":           string(fixture) + "\n",
		"no trailing newline":  strings.TrimSuffix(string(fixture), "\n"),
		"empty":                "",
	} {
		want, wantErr := postiReference([]byte(source))
		got, gotErr := postiLibrary([]byte(source))
		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("%s: library error = %v, reference error = %v", name, gotErr, wantErr)
		}
		if wantErr == nil && !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: library rows = %q, reference rows = %q", name, got, want)
		}
		if gotErr != nil && !errors.Is(gotErr, ErrorMalformedRow) {
			t.Fatalf("%s: library error = %v, want malformed row", name, gotErr)
		}
	}
	rows, err := postiLibrary(fixture)
	if err != nil || len(rows) != 2 || rows[0][1] != "HÄMEENLINNA" || rows[0][3] != "Kanta-Häme" {
		t.Fatalf("posti fixture rows = %q, %v", rows, err)
	}
}

func TestDelimitedLineRecordsMatchPostalGeposit(t *testing.T) {
	t.Parallel()

	fixture := string(readParityFixture(t, "geposit.csv"))
	for name, source := range map[string]string{
		"fixture":                    fixture,
		"crlf":                       strings.ReplaceAll(fixture, "\n", "\r\n"),
		"no trailing newline":        strings.TrimSuffix(fixture, "\n"),
		"quoted field":               fixture + "\"12354\";\"Stockholm\";S;Central;\"Quoted; Street\";1;1;;\n",
		"quoted field across lines":  fixture + "12355;Stockholm;S;Central;\"Two\nLines\";1;1;;\n",
		"blank line between records": strings.Replace(fixture, "\n", "\n\n", 2),
		"crlf blank line":            strings.Replace(strings.ReplaceAll(fixture, "\n", "\r\n"), "\r\n", "\r\n\r\n", 2),
		"trailing blank line":        fixture + "\n",
		"leading blank line":         "\n" + fixture,
		"whitespace-only line":       fixture + " \n",
		"bare quote":                 fixture + "12356;Stock\"holm;S;Central;;;;;\n",
		"empty":                      "",
	} {
		want, wantErr := gepositReference([]byte(source))
		got, gotErr := gepositLibrary([]byte(source))
		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("%s: library error = %v, reference error = %v", name, gotErr, wantErr)
		}
		if wantErr == nil && !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: library rows = %q, reference rows = %q", name, got, want)
		}
		if gotErr != nil && !errors.Is(gotErr, ErrorMalformedRow) {
			t.Fatalf("%s: library error = %v, want malformed row", name, gotErr)
		}
	}
	rows, err := gepositLibrary([]byte(fixture))
	if err != nil || len(rows) != 11 || rows[9][4] != "Nordic Letters" || rows[9][7] != "ä" {
		t.Fatalf("geposit fixture rows = %q, %v", rows, err)
	}
}

func TestLineRecordOptionsReportPhysicalLines(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		source string
		config DelimitedConfig
		row    int
	}{
		"blank line":          {"a;b\n\nc;d\n", DelimitedConfig{Delimiter: ';', RejectBlankLines: true}, 2},
		"trailing blank":      {"a;b\nc;d\n\n", DelimitedConfig{Delimiter: ';', RejectBlankLines: true}, 3},
		"spanning record":     {"a;b\n\"c\nd\";e\n", DelimitedConfig{Delimiter: ';', SingleLineRecords: true}, 2},
		"spanning last field": {"a;\"b\nc\"\n", DelimitedConfig{Delimiter: ';', SingleLineRecords: true}, 1},
	} {
		reader, err := NewDelimitedReader(strings.NewReader(testCase.source), testCase.config)
		if err != nil {
			t.Fatal(err)
		}
		for {
			_, err = reader.Read()
			if err != nil {
				break
			}
		}
		var tabularErr *Error
		if !errors.As(err, &tabularErr) || !errors.Is(err, ErrorMalformedRow) || tabularErr.Row != testCase.row {
			t.Fatalf("%s: Read() error = %#v, want malformed row %d", name, err, testCase.row)
		}
	}
	// Blank lines inside a quoted field are field content, not blank lines.
	reader, err := NewDelimitedReader(strings.NewReader("a;\"b\n\nc\"\nd;e\n"), DelimitedConfig{
		Delimiter: ';', RejectBlankLines: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []Row{{"a", "b\n\nc"}, {"d", "e"}} {
		row, readErr := reader.Read()
		if readErr != nil || !reflect.DeepEqual(row, want) {
			t.Fatalf("Read() = %q, %v; want %q", row, readErr, want)
		}
	}
	if _, err = reader.Read(); !errors.Is(err, io.EOF) {
		t.Fatalf("Read() after quoted blank line = %v, want EOF", err)
	}
}

func TestNewOptionsKeepZeroValueBehaviour(t *testing.T) {
	t.Parallel()

	reader, err := NewDelimitedReader(strings.NewReader("\na;\"b\nc\"\n\n"), DelimitedConfig{Delimiter: ';'})
	if err != nil {
		t.Fatal(err)
	}
	row, err := reader.Read()
	if err != nil || !reflect.DeepEqual(row, Row{"a", "b\nc"}) {
		t.Fatalf("default delimited Read() = %q, %v", row, err)
	}
	if _, err = reader.Read(); !errors.Is(err, io.EOF) {
		t.Fatalf("default delimited trailing blank = %v, want EOF", err)
	}

	fixed, err := NewFixedWidthReader(strings.NewReader("ab \x00cd\n"), FixedWidthConfig{
		Fields: []FixedWidthField{{Name: "a", Start: 0, End: 4}},
	})
	if err != nil {
		t.Fatal(err)
	}
	row, err = fixed.Read()
	if err != nil || !reflect.DeepEqual(row, Row{"ab \x00"}) {
		t.Fatalf("default fixed-width Read() = %q, %v", row, err)
	}
}

func TestNewOptionsRejectContradictoryConfiguration(t *testing.T) {
	t.Parallel()

	field := []FixedWidthField{{Name: "a", Start: 0, End: 4}}
	for name, config := range map[string]FixedWidthConfig{
		"zero width":          {Fields: field, RecordWidths: []int{0}},
		"width below layout":  {Fields: field, RecordWidths: []int{3}},
		"duplicate width":     {Fields: field, RecordWidths: []int{4, 4}},
		"with short records":  {Fields: field, RecordWidths: []int{4}, AllowShortRecords: true},
		"with trailing check": {Fields: field, RecordWidths: []int{4}, RejectTrailingBytes: true},
		"cutset and trim space": {Fields: []FixedWidthField{{
			Name: "a", Start: 0, End: 4, TrimSpace: true, TrimCutset: " ",
		}}},
		"cutset not UTF-8": {Fields: []FixedWidthField{{Name: "a", Start: 0, End: 4, TrimCutset: "\xff"}}},
	} {
		if _, err := NewFixedWidthReader(strings.NewReader(""), config); !errors.Is(err, ErrorInvalidLayout) {
			t.Fatalf("%s: NewFixedWidthReader() error = %v, want invalid layout", name, err)
		}
	}
	if _, err := NewDelimitedReader(strings.NewReader(""), DelimitedConfig{
		Delimiter: ';', Comment: '#', RejectBlankLines: true,
	}); !errors.Is(err, ErrorInvalidConfig) {
		t.Fatalf("NewDelimitedReader(comment with blank-line rejection) error = %v", err)
	}
}

func TestRecordWidthsCopiesCallerSlice(t *testing.T) {
	t.Parallel()

	widths := []int{4}
	reader, err := NewFixedWidthReader(strings.NewReader("abcd\nabcde\n"), FixedWidthConfig{
		Fields: []FixedWidthField{{Name: "a", Start: 0, End: 4}}, RecordWidths: widths,
	})
	if err != nil {
		t.Fatal(err)
	}
	widths[0] = 5
	if row, readErr := reader.Read(); readErr != nil || row[0] != "abcd" {
		t.Fatalf("Read() = %q, %v", row, readErr)
	}
	if _, readErr := reader.Read(); !errors.Is(readErr, ErrorMalformedRow) {
		t.Fatalf("Read(width 5) = %v, want malformed row", readErr)
	}
}

// FuzzPostalLineRecords checks that the line options reject and accept
// exactly what Postal's line-at-a-time Geposit framing does for valid UTF-8.
func FuzzPostalLineRecords(f *testing.F) {
	f.Add([]byte("a;b\n\"c;d\";e\n"))
	f.Add([]byte("a;b\n\nc\n"))
	f.Add([]byte("a;\"b\nc\"\n"))
	f.Add([]byte("a;b\r\n\r\nc\r\n"))
	f.Fuzz(func(t *testing.T, source []byte) {
		// Postal trims one carriage return before encoding/csv trims another,
		// so "\r\r" before a line end differs by one data byte. That double
		// trim is a Postal quirk the library does not reproduce.
		if !utf8.Valid(source) || len(source) > 4096 || bytes.Contains(source, []byte("\r\r")) {
			t.Skip()
		}
		want, wantErr := gepositReference(source)
		got, gotErr := gepositLibrary(source)
		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("library error = %v, reference error = %v for %q", gotErr, wantErr, source)
		}
		if wantErr == nil && !reflect.DeepEqual(got, want) {
			t.Fatalf("library rows = %q, reference rows = %q for %q", got, want, source)
		}
	})
}

// FuzzPostalFixedWidth checks exact record widths and cutset trimming against
// Postal's Posti framing.
func FuzzPostalFixedWidth(f *testing.F) {
	line := strings.Repeat(" ", 13) + "00100HELSINKI" + strings.Repeat("\x00", 158)
	f.Add([]byte(line + "\n"))
	f.Add([]byte(line + strings.Repeat("\v", 44) + "\r\n"))
	f.Add([]byte(line[:175] + "\n"))
	f.Fuzz(func(t *testing.T, source []byte) {
		want, wantErr := postiReference(source)
		got, gotErr := postiLibrary(source)
		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("library error = %v, reference error = %v", gotErr, wantErr)
		}
		if wantErr == nil && !reflect.DeepEqual(got, want) {
			t.Fatalf("library rows = %q, reference rows = %q", got, want)
		}
	})
}
