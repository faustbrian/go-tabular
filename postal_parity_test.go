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
// span lines are malformed, and a line is read through a 4,098-byte buffer
// and refused above 4,096 bytes once its line break is removed.
func gepositReference(source []byte) ([]Row, error) {
	reader := bufio.NewReaderSize(bytes.NewReader(source), gepositRowBytes+2)
	var rows []Row
	for {
		line, err := reader.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			return rows, errors.New("row exceeds its byte limit")
		}
		if len(line) == 0 && errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return rows, err
		}
		line = bytes.TrimSuffix(bytes.TrimSuffix(line, []byte{'\n'}), []byte{'\r'})
		if len(line) > gepositRowBytes {
			return rows, errors.New("row exceeds its byte limit")
		}
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

// Postal's Geposit row and source bounds. The library's record bound counts
// the line break, so MaxRecordBytes 4098 admits up to 4,097 content bytes
// before LF and 4,098 before end of input, one and two more than Postal's
// 4,096. That looseness in a safety bound is accepted.
const (
	gepositRowBytes    = 4096
	gepositSourceBytes = 256 << 20
)

func gepositLibrary(source []byte) ([]Row, error) {
	reader, err := NewDelimitedReader(bytes.NewReader(source), DelimitedConfig{
		Delimiter: ';', AllowVariableFields: true, SingleLineRecords: true, RejectBlankLines: true,
		MaxRecordBytes: gepositRowBytes + 2, MaxSourceBytes: gepositSourceBytes,
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
		"cutset not UTF-8":           {Fields: []FixedWidthField{{Name: "a", Start: 0, End: 4, TrimCutset: "\xff"}}},
		"width beyond record bound":  {Fields: field, RecordWidths: []int{4, 7}, MaxRecordBytes: 8},
		"width beyond default bound": {Fields: field, RecordWidths: []int{1<<20 - 1}},
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

// A CRLF-framed record needs its width plus two bytes of Scanner buffer, so
// every admitted width must fit MaxRecordBytes with that framing.
func TestRecordWidthsFitTheRecordBoundWithCRLF(t *testing.T) {
	t.Parallel()

	reader, err := NewFixedWidthReader(strings.NewReader("abcdef\r\nabcd\r\nabcdef"), FixedWidthConfig{
		Fields: []FixedWidthField{{Name: "a", Start: 0, End: 4}}, RecordWidths: []int{4, 6}, MaxRecordBytes: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if row, readErr := reader.Read(); readErr != nil || row[0] != "abcd" {
			t.Fatalf("Read() = %q, %v", row, readErr)
		}
	}
	if _, err := reader.Read(); !errors.Is(err, io.EOF) {
		t.Fatalf("Read() error = %v, want EOF", err)
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

// hasKnownGepositDelta reports the two accepted differences from Postal's
// framing: a line ending in "\r\r", where Postal strips a carriage return
// before encoding/csv strips another, and a line of 4,097 or 4,098 content
// bytes, which only the library's one- or two-byte looser bound admits.
func hasKnownGepositDelta(source []byte) bool {
	for _, line := range bytes.SplitAfter(source, []byte{'\n'}) {
		content := bytes.TrimSuffix(line, []byte{'\n'})
		if bytes.HasSuffix(content, []byte("\r\r")) {
			return true
		}
		if length := len(bytes.TrimSuffix(content, []byte{'\r'})); length > gepositRowBytes && length <= gepositRowBytes+2 {
			return true
		}
	}
	return false
}

// A doubled carriage return before a line end loses one CR in Postal but
// keeps it as field data in the library. The changed field can then fail the
// importer's validation and the whole import, so this delta is accepted only
// because it fails closed on corrupted line endings.
func TestKnownGepositCarriageReturnDelta(t *testing.T) {
	t.Parallel()

	for _, source := range []string{"0\r\r\n", "0\r\r"} {
		want, wantErr := gepositReference([]byte(source))
		got, gotErr := gepositLibrary([]byte(source))
		if wantErr != nil || gotErr != nil || !reflect.DeepEqual(want, []Row{{"0"}}) ||
			!reflect.DeepEqual(got, []Row{{"0\r"}}) {
			t.Fatalf("%q: reference = %q, %v; library = %q, %v", source, want, wantErr, got, gotErr)
		}
	}
}

// Postal admits 4,096 content bytes per row. The library agrees at that
// bound and beyond the two-byte framing allowance, and admits the accepted
// one- or two-byte looser rows in between.
func TestGepositRowBoundMatchesPostal(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		length int
		ending string
		delta  bool
	}{
		{gepositRowBytes, "", false},
		{gepositRowBytes, "\n", false},
		{gepositRowBytes, "\r\n", false},
		{gepositRowBytes + 1, "", true},
		{gepositRowBytes + 1, "\n", true},
		{gepositRowBytes + 1, "\r\n", false},
		{gepositRowBytes + 2, "", true},
		{gepositRowBytes + 2, "\n", false},
		{gepositRowBytes + 3, "", false},
	} {
		source := []byte(strings.Repeat("x", testCase.length) + testCase.ending)
		if hasKnownGepositDelta(source) != (testCase.length > gepositRowBytes && testCase.length <= gepositRowBytes+2) {
			t.Fatalf("%d bytes + %q: hasKnownGepositDelta disagrees with the documented delta", testCase.length, testCase.ending)
		}
		_, wantErr := gepositReference(source)
		_, gotErr := gepositLibrary(source)
		if (wantErr == nil) != (testCase.length <= gepositRowBytes) {
			t.Fatalf("%d bytes + %q: reference error = %v", testCase.length, testCase.ending, wantErr)
		}
		if (gotErr == nil) != (testCase.length <= gepositRowBytes || testCase.delta) {
			t.Fatalf("%d bytes + %q: library error = %v", testCase.length, testCase.ending, gotErr)
		}
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
		if !utf8.Valid(source) || len(source) > 16<<10 || hasKnownGepositDelta(source) {
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

// readAllRecovering keeps reading past row errors, as an import that logs and
// skips bad rows would, and returns every row and every error line.
func readAllRecovering(t *testing.T, reader *DelimitedReader) ([]Row, []int) {
	t.Helper()
	var rows []Row
	var errorLines []int
	for range 100 {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return rows, errorLines
		}
		if err != nil {
			var tabularErr *Error
			if !errors.As(err, &tabularErr) ||
				!errors.Is(err, ErrorMalformedRow) && !errors.Is(err, ErrorLimitExceeded) {
				t.Fatalf("Read() error = %#v, want a recoverable row error", err)
			}
			errorLines = append(errorLines, tabularErr.Row)
			continue
		}
		rows = append(rows, row)
	}
	t.Fatal("Read() never returned io.EOF")
	return nil, nil
}

func TestRejectBlankLinesRecoversAfterEveryError(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		source     string
		config     DelimitedConfig
		rows       []Row
		errorLines []int
	}{
		"field count error": {
			"a;b\nc\nd;e\n", DelimitedConfig{Delimiter: ';', RejectBlankLines: true},
			[]Row{{"a", "b"}, {"d", "e"}}, []int{2},
		},
		"blank lines between records": {
			"a\n\nb\nc\n", DelimitedConfig{Delimiter: ';', RejectBlankLines: true},
			[]Row{{"a"}, {"b"}, {"c"}}, []int{2},
		},
		"blank run reported once": {
			"a\n\n\nb\n", DelimitedConfig{Delimiter: ';', RejectBlankLines: true},
			[]Row{{"a"}, {"b"}}, []int{2},
		},
		"trailing blank lines": {
			"a\nb\n\n\n", DelimitedConfig{Delimiter: ';', RejectBlankLines: true},
			[]Row{{"a"}, {"b"}}, []int{3},
		},
		"blank line before a syntax error": {
			"a\n\nb\"c\nd\n", DelimitedConfig{Delimiter: ';', RejectBlankLines: true},
			[]Row{{"a"}, {"d"}}, []int{2, 3},
		},
		"field limit": {
			"a\nlong\n\nb\n", DelimitedConfig{Delimiter: ';', RejectBlankLines: true, MaxFieldBytes: 3},
			[]Row{{"a"}, {"b"}}, []int{2, 3},
		},
		"spanning record": {
			"a\n\"b\nc\"\n\nd\n", DelimitedConfig{Delimiter: ';', RejectBlankLines: true, SingleLineRecords: true},
			[]Row{{"a"}, {"d"}}, []int{2, 4},
		},
	} {
		reader, err := NewDelimitedReader(strings.NewReader(testCase.source), testCase.config)
		if err != nil {
			t.Fatal(err)
		}
		rows, errorLines := readAllRecovering(t, reader)
		if !reflect.DeepEqual(rows, testCase.rows) || !reflect.DeepEqual(errorLines, testCase.errorLines) {
			t.Fatalf("%s: rows = %q, error lines = %v; want %q, %v",
				name, rows, errorLines, testCase.rows, testCase.errorLines)
		}
	}
}

func TestLineRecordOptionsKeepSourceLimitErrors(t *testing.T) {
	t.Parallel()

	reader, err := NewDelimitedReader(strings.NewReader("a\n"+strings.Repeat("b", 64)+"\n"), DelimitedConfig{
		Delimiter: ';', RejectBlankLines: true, SingleLineRecords: true, MaxRecordBytes: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	if row, readErr := reader.Read(); readErr != nil || !reflect.DeepEqual(row, Row{"a"}) {
		t.Fatalf("Read() = %q, %v", row, readErr)
	}
	if _, err = reader.Read(); !errors.Is(err, ErrorLimitExceeded) {
		t.Fatalf("Read(over record limit) = %v, want limit exceeded", err)
	}
}
