package tabular

import (
	"bufio"
	"errors"
	"io"
	"slices"
	"strings"
	"unicode/utf8"
)

const defaultMaxRecordBytes = 1024 * 1024

// FixedWidthField identifies a half-open byte range [Start, End) in a source
// record. Offsets apply before character decoding.
type FixedWidthField struct {
	// Name is a required nonempty output label; duplicate labels are not
	// rejected.
	Name string
	// Start is a zero-based original-byte offset, before decoding; gaps are
	// allowed.
	Start int
	// End is the exclusive original-byte end, greater than Start. Fields must
	// be ordered and nonoverlapping.
	End int
	// TrimSpace trims the decoded field using strings.TrimSpace before row
	// normalization.
	TrimSpace bool
	// TrimCutset, when nonempty, trims every leading and trailing character in
	// the set from the decoded field using strings.Trim before row
	// normalization, for example " \t\n\r\x00\v" to also remove NUL padding.
	// It must be valid UTF-8 and cannot be combined with TrimSpace; empty
	// keeps the field untrimmed.
	TrimCutset string
}

// FixedWidthConfig controls fixed-width record parsing.
type FixedWidthConfig struct {
	// Fields defines a nonempty ordered layout; construction copies the slice.
	Fields []FixedWidthField
	// Encoding selects explicit field decoding; empty selects validated UTF-8.
	Encoding Encoding
	// AllowShortRecords clips fields to available bytes; wholly missing ranges
	// decode as empty strings instead of failing record width.
	AllowShortRecords bool
	// RejectTrailingBytes rejects bytes beyond the final End; false ignores
	// them.
	RejectTrailingBytes bool
	// RecordWidths, when nonempty, is the exact set of admitted record byte
	// lengths after line framing; any other length is a malformed row. Every
	// width must be unique and at least the final End, and the option cannot
	// be combined with AllowShortRecords or RejectTrailingBytes. Construction
	// copies the slice; empty keeps the layout-width checks above.
	RecordWidths []int
	// MaxRecordBytes sets the Scanner buffer/token ceiling; zero selects 1 MiB
	// and negative values fail. Line framing may need space beyond data bytes,
	// so payload equality is not an inclusive admission promise.
	MaxRecordBytes int
	// Normalize transforms decoded fields after their individual TrimSpace
	// options.
	Normalize NormalizationConfig
}

// FixedWidthReader streams newline-delimited fixed-width records.
type FixedWidthReader struct {
	scanner             *bufio.Scanner
	fields              []FixedWidthField
	encoding            Encoding
	allowShortRecords   bool
	rejectTrailingBytes bool
	recordWidths        []int
	normalize           NormalizationConfig
	recordWidth         int
	row                 int
}

// NewFixedWidthReader validates and copies a layout without reading source.
// It retains but never closes source, scans newline records with ScanLines, and
// decodes each selected byte range separately. A UTF-8 range splitting a rune
// fails decoding rather than silently repairing the boundary.
func NewFixedWidthReader(source io.Reader, config FixedWidthConfig) (*FixedWidthReader, error) {
	if source == nil || len(config.Fields) == 0 || config.MaxRecordBytes < 0 {
		return nil, &Error{Kind: ErrorInvalidLayout, Op: "fixedwidth.new", Format: "fixed-width"}
	}
	if config.Encoding == "" {
		config.Encoding = EncodingUTF8
	}
	if _, err := decoderFor(config.Encoding); err != nil {
		return nil, err
	}

	fields := append([]FixedWidthField(nil), config.Fields...)
	end := 0
	for index, field := range fields {
		if field.Name == "" || field.Start < 0 || field.End <= field.Start || field.Start < end ||
			(field.TrimCutset != "" && (field.TrimSpace || !utf8.ValidString(field.TrimCutset))) {
			return nil, &Error{
				Kind:   ErrorInvalidLayout,
				Op:     "fixedwidth.new",
				Format: "fixed-width",
				Field:  index + 1,
			}
		}
		end = field.End
	}
	recordWidths, ok := admittedRecordWidths(config, end)
	if !ok {
		return nil, &Error{Kind: ErrorInvalidLayout, Op: "fixedwidth.new", Format: "fixed-width"}
	}

	maxRecordBytes := config.MaxRecordBytes
	if maxRecordBytes == 0 {
		maxRecordBytes = defaultMaxRecordBytes
	}
	scanner := bufio.NewScanner(source)
	scanner.Buffer(nil, maxRecordBytes)

	return &FixedWidthReader{
		scanner:             scanner,
		fields:              fields,
		encoding:            config.Encoding,
		allowShortRecords:   config.AllowShortRecords,
		rejectTrailingBytes: config.RejectTrailingBytes,
		recordWidths:        recordWidths,
		normalize:           config.Normalize,
		recordWidth:         end,
	}, nil
}

// Fields returns an independent slice of configured names in source order.
func (reader *FixedWidthReader) Fields() []string {
	names := make([]string, len(reader.fields))
	for index, field := range reader.fields {
		names[index] = field.Name
	}
	return names
}

// Read returns an owned decoded/normalized row. io.EOF marks a clean end.
// All Scanner failures, including source I/O errors, are classified as
// ErrorLimitExceeded with the original cause; shape/decoding failures return
// no partial row. The zero reader value is not usable.
func (reader *FixedWidthReader) Read() (Row, error) {
	if !reader.scanner.Scan() {
		if err := reader.scanner.Err(); err != nil {
			return nil, &Error{
				Kind:   ErrorLimitExceeded,
				Op:     "fixedwidth.read",
				Format: "fixed-width",
				Row:    reader.row + 1,
				Err:    err,
			}
		}
		return nil, io.EOF
	}
	reader.row++
	record := reader.scanner.Bytes()
	if len(reader.recordWidths) > 0 && !slices.Contains(reader.recordWidths, len(record)) {
		return nil, reader.rowError(ErrorMalformedRow, 0, errors.New("record width is not admitted"))
	}
	if len(record) < reader.recordWidth && !reader.allowShortRecords {
		return nil, reader.rowError(ErrorMalformedRow, 0, errors.New("record is shorter than layout"))
	}
	if len(record) > reader.recordWidth && reader.rejectTrailingBytes {
		return nil, reader.rowError(ErrorMalformedRow, 0, errors.New("record has trailing bytes"))
	}

	row := make(Row, len(reader.fields))
	for index, field := range reader.fields {
		start := min(field.Start, len(record))
		end := min(field.End, len(record))
		value, err := DecodeBytes(record[start:end], reader.encoding)
		if err != nil {
			return nil, reader.rowError(ErrorInvalidEncoding, index+1, err)
		}
		if field.TrimSpace {
			value = strings.TrimSpace(value)
		}
		if field.TrimCutset != "" {
			value = strings.Trim(value, field.TrimCutset)
		}
		row[index] = value
	}
	return NormalizeRow(row, reader.normalize), nil
}

// admittedRecordWidths validates and copies RecordWidths against the layout.
func admittedRecordWidths(config FixedWidthConfig, layoutEnd int) ([]int, bool) {
	if len(config.RecordWidths) == 0 {
		return nil, true
	}
	if config.AllowShortRecords || config.RejectTrailingBytes {
		return nil, false
	}
	widths := make([]int, 0, len(config.RecordWidths))
	for _, width := range config.RecordWidths {
		if width < layoutEnd || slices.Contains(widths, width) {
			return nil, false
		}
		widths = append(widths, width)
	}
	return widths, true
}

func (reader *FixedWidthReader) rowError(kind ErrorKind, field int, cause error) error {
	return &Error{
		Kind:   kind,
		Op:     "fixedwidth.read",
		Format: "fixed-width",
		Row:    reader.row,
		Field:  field,
		Err:    cause,
	}
}

// ExtractBytes returns a nonempty half-open byte range without copying. The
// result aliases record and retains its backing array; mutations are shared.
// Invalid or empty ranges return ErrorInvalidLayout; no decoding is performed.
func ExtractBytes(record []byte, start, end int) ([]byte, error) {
	if start < 0 || end <= start || end > len(record) {
		return nil, &Error{Kind: ErrorInvalidLayout, Op: "fixedwidth.extract", Format: "fixed-width"}
	}
	return record[start:end], nil
}
