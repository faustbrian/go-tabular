package tabular

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

const delimitedReadBufferSize = 64 * 1024

var errDelimitedRecordLimit = errors.New("delimited record exceeds byte limit")
var errDelimitedSourceLimit = errors.New("delimited source exceeds byte limit")

const defaultMaxDelimitedSourceBytes int64 = 64 * 1024 * 1024

// DelimitedConfig explicitly controls delimited-text parsing behavior.
type DelimitedConfig struct {
	// Delimiter selects the explicit separator for NewDelimitedReader.
	// NewCSVReader overrides it with comma; no delimiter is inferred.
	Delimiter rune
	// Comment selects a comment marker only at line start; zero disables
	// comments. It must differ from Delimiter.
	Comment rune
	// LazyQuotes opts into encoding/csv's permissive quoting behavior; false
	// keeps its strict quote syntax.
	LazyQuotes bool
	// TrimLeadingSpace applies encoding/csv's leading Unicode-space handling to
	// fields before record return, not trailing trim.
	TrimLeadingSpace bool
	// AllowVariableFields disables field-count validation, overriding
	// FieldsPerRecord.
	AllowVariableFields bool
	// FieldsPerRecord is a nonnegative expected width. Zero learns the first
	// parsed record's width (including a configured header) unless variable
	// mode is enabled.
	FieldsPerRecord int
	// MaxRecordBytes bounds one logical record before parser allocation.
	// Zero selects a finite 1 MiB default.
	MaxRecordBytes int
	// MaxFieldBytes bounds one parsed field before normalization.
	// Zero selects a finite 1 MiB default.
	MaxFieldBytes int
	// MaxSourceBytes bounds all bytes consumed, including headers, blank lines and
	// comments, before CSV parsing. Zero selects 64 MiB; positive values override it.
	MaxSourceBytes int64
	// Header enables consuming the first parsed record as a header.
	// Construction copies the pointed value and replacement map; nil disables
	// it.
	Header *HeaderConfig
	// Normalize applies TrimSpace then EmptyAs to data rows after parsed-field
	// admission; header normalization is separate.
	Normalize NormalizationConfig
	// SingleLineRecords requires every record, including a configured header,
	// to occupy exactly one physical line: a quoted field containing a line
	// break is a malformed row. False keeps encoding/csv's multiline fields.
	SingleLineRecords bool
	// RejectBlankLines makes an empty physical line before, between or after
	// records a malformed row reported at that line, instead of encoding/csv's
	// silent skip. Line breaks inside quoted fields are field content. It
	// cannot be combined with Comment, whose skipped lines are indistinguishable.
	RejectBlankLines bool
}

// DelimitedReader streams records from CSV or another delimited text format.
type DelimitedReader struct {
	reader        *csv.Reader
	format        string
	header        Row
	headerConfig  *HeaderConfig
	headerRead    bool
	headerErr     error
	normalize     NormalizationConfig
	maxFieldBytes int
	row           int
	singleLine    bool
	rejectBlank   bool
	source        *delimitedSourceLimitReader
	// lastLine is the last physical line already returned or reported.
	lastLine int
	// pending holds a parsed record or error withheld while the blank lines
	// before it are reported.
	pending *pendingRecord
}

// pendingRecord is one encoding/csv result with the physical lines it spans.
type pendingRecord struct {
	record []string
	err    error
	start  int
	end    int
}

// NewCSVReader constructs a comma-delimited streaming reader without reading
// source. It borrows source, copies mutable header configuration, and overrides
// config.Delimiter. Character decoding/validation must be configured upstream.
func NewCSVReader(source io.Reader, config DelimitedConfig) (*DelimitedReader, error) {
	config.Delimiter = ','
	reader, err := newDelimitedReader(source, config, "csv")
	if err != nil {
		return nil, err
	}
	return reader, nil
}

// NewDelimitedReader constructs a reader for an explicit delimiter without
// reading or taking source ownership. Its grammar follows Go encoding/csv
// (TABULAR-DEC-001), not strict RFC 4180 conformance. No encoding is inferred.
func NewDelimitedReader(source io.Reader, config DelimitedConfig) (*DelimitedReader, error) {
	return newDelimitedReader(source, config, "delimited")
}

func newDelimitedReader(source io.Reader, config DelimitedConfig, format string) (*DelimitedReader, error) {
	if source == nil || !validDelimiter(config.Delimiter) ||
		(config.Comment != 0 && (!validDelimiter(config.Comment) || config.Comment == config.Delimiter)) ||
		config.FieldsPerRecord < 0 || config.MaxRecordBytes < 0 ||
		config.MaxFieldBytes < 0 || config.MaxSourceBytes < 0 ||
		(config.RejectBlankLines && config.Comment != 0) {
		return nil, &Error{Kind: ErrorInvalidConfig, Op: "delimited.new", Format: format}
	}

	maxRecordBytes := config.MaxRecordBytes
	if maxRecordBytes == 0 {
		maxRecordBytes = defaultMaxRecordBytes
	}
	maxFieldBytes := config.MaxFieldBytes
	if maxFieldBytes == 0 {
		maxFieldBytes = defaultMaxRecordBytes
	}
	maxSourceBytes := config.MaxSourceBytes
	if maxSourceBytes == 0 {
		maxSourceBytes = defaultMaxDelimitedSourceBytes
	}
	sourceLimit := &delimitedSourceLimitReader{
		source: source, remaining: maxSourceBytes, countLines: config.RejectBlankLines,
	}
	parserSource := &delimitedRecordLimitReader{
		source:       sourceLimit,
		maximum:      maxRecordBytes,
		trimLeading:  config.TrimLeadingSpace,
		delimiter:    []byte(string(config.Delimiter)),
		atFieldStart: true,
		comment:      []byte(string(config.Comment)),
		lineStart:    true,
	}
	if config.Comment == 0 {
		parserSource.comment = nil
	}
	parser := csv.NewReader(bufio.NewReaderSize(parserSource, delimitedReadBufferSize))
	parser.Comma = config.Delimiter
	parser.Comment = config.Comment
	parser.LazyQuotes = config.LazyQuotes
	parser.TrimLeadingSpace = config.TrimLeadingSpace
	parser.FieldsPerRecord = config.FieldsPerRecord
	if config.AllowVariableFields {
		parser.FieldsPerRecord = -1
	}

	return &DelimitedReader{
		reader:        parser,
		format:        format,
		headerConfig:  cloneHeaderConfig(config.Header),
		normalize:     config.Normalize,
		maxFieldBytes: maxFieldBytes,
		singleLine:    config.SingleLineRecords,
		rejectBlank:   config.RejectBlankLines,
		source:        sourceLimit,
	}, nil
}

// Header returns the normalized first row when header handling is configured.
// The returned row is a copy and is safe for the caller to modify. Nil header
// configuration consumes nothing and returns nil, nil. The first header/error
// is cached, including a missing-header ErrorInvalidHeader wrapping io.EOF.
func (reader *DelimitedReader) Header() (Row, error) {
	if reader.headerConfig == nil {
		return nil, nil
	}
	reader.readHeader()
	if reader.headerErr != nil {
		return nil, reader.headerErr
	}
	return append(Row(nil), reader.header...), nil
}

// Read returns an independently owned next record after configured header
// consumption. io.EOF marks a clean end; parser/source errors discard partial
// rows and retain their cause in a typed Error. Coordinates may be physical CSV
// lines on syntax errors. Previously delivered rows are never rolled back.
func (reader *DelimitedReader) Read() (Row, error) {
	if reader.headerConfig != nil {
		reader.readHeader()
		if reader.headerErr != nil {
			return nil, reader.headerErr
		}
	}

	record, err := reader.readRecord()
	if err != nil {
		return nil, err
	}
	if !reader.normalize.TrimSpace && reader.normalize.EmptyAs == "" {
		return record, nil
	}
	return NormalizeRow(record, reader.normalize), nil
}

func (reader *DelimitedReader) readHeader() {
	if reader.headerRead {
		return
	}
	reader.headerRead = true

	header, err := reader.readRecord()
	if err != nil {
		if errors.Is(err, io.EOF) {
			reader.headerErr = &Error{
				Kind:   ErrorInvalidHeader,
				Op:     "delimited.header",
				Format: reader.format,
				Err:    io.EOF,
			}
			return
		}
		reader.headerErr = err
		return
	}
	reader.header, reader.headerErr = NormalizeHeader(header, *reader.headerConfig)
}

func (reader *DelimitedReader) readRecord() (Row, error) {
	if !reader.singleLine && !reader.rejectBlank {
		record, err := reader.reader.Read()
		return reader.admit(record, err)
	}
	next := reader.pending
	if next == nil {
		next = reader.readPositioned()
	}
	// Blank lines are reported once per run, at the run's first line; the
	// record after them is withheld and returned by the next Read.
	if reader.rejectBlank && next.start > reader.lastLine+1 {
		reader.pending = next
		blank := reader.lastLine + 1
		reader.lastLine = next.start - 1
		return nil, reader.malformedLine(blank, errors.New("blank line"))
	}
	reader.pending = nil
	if errors.Is(next.err, io.EOF) {
		return nil, io.EOF
	}
	reader.lastLine = next.end
	if next.err == nil && reader.singleLine && next.end != next.start {
		reader.row++
		return nil, reader.malformedLine(next.start, errors.New("record spans lines"))
	}
	return reader.admit(next.record, next.err)
}

// readPositioned reads one encoding/csv result with the physical lines it
// consumed. At the end of input the remaining lines are blank, so EOF is
// positioned after them.
func (reader *DelimitedReader) readPositioned() *pendingRecord {
	record, err := reader.reader.Read()
	next := &pendingRecord{record: record, err: err}
	var parseErr *csv.ParseError
	switch {
	case err == nil:
		next.start, _ = reader.reader.FieldPos(0)
		last := len(record) - 1
		next.end, _ = reader.reader.FieldPos(last)
		next.end += strings.Count(record[last], "\n")
	case errors.Is(err, io.EOF):
		next.start = reader.source.lines() + 1
		next.end = next.start - 1
	case errors.As(err, &parseErr):
		next.start, next.end = parseErr.StartLine, parseErr.Line
	default:
		next.start, next.end = reader.lastLine+1, reader.lastLine+1
	}
	return next
}

// admit classifies one encoding/csv result as the reader's row or error.
func (reader *DelimitedReader) admit(record []string, err error) (Row, error) {
	if err == nil {
		reader.row++
		if limitErr := reader.fieldLimitError(record, reader.row); limitErr != nil {
			return nil, limitErr
		}
		return Row(record), nil
	}
	if errors.Is(err, io.EOF) {
		return nil, io.EOF
	}

	row := reader.row + 1
	var parseErr *csv.ParseError
	if errors.As(err, &parseErr) {
		row = parseErr.Line
	}
	if limitErr := reader.fieldLimitError(record, row); limitErr != nil {
		return nil, limitErr
	}
	kind := ErrorMalformedRow
	if errors.Is(err, errDelimitedRecordLimit) || errors.Is(err, errDelimitedSourceLimit) {
		kind = ErrorLimitExceeded
	}
	return nil, &Error{
		Kind:   kind,
		Op:     "delimited.read",
		Format: reader.format,
		Row:    row,
		Err:    err,
	}
}

func (reader *DelimitedReader) malformedLine(line int, cause error) error {
	return &Error{Kind: ErrorMalformedRow, Op: "delimited.read", Format: reader.format, Row: line, Err: cause}
}

func (reader *DelimitedReader) fieldLimitError(record []string, row int) error {
	for index, field := range record {
		if len(field) > reader.maxFieldBytes {
			return &Error{
				Kind:   ErrorLimitExceeded,
				Op:     "delimited.read",
				Format: reader.format,
				Row:    row,
				Field:  index + 1,
			}
		}
	}
	return nil
}

// This observer bounds bytes delivered to encoding/csv; it is not a replacement
// grammar parser. Quote/delimiter/comment state must survive arbitrary source
// chunk boundaries so multiline records cannot reset the budget prematurely.
type delimitedRecordLimitReader struct {
	source           io.Reader
	maximum          int
	recordBytes      int
	inQuotes         bool
	quotePending     bool
	quoteCarriage    bool
	quoteDelimiterAt int
	limitErr         error
	trimLeading      bool
	delimiter        []byte
	delimiterAt      int
	atFieldStart     bool
	leadingBytes     [utf8.UTFMax]byte
	leadingBytesAt   int
	comment          []byte
	commentAt        int
	lineStart        bool
	inComment        bool
}

type delimitedSourceLimitReader struct {
	source    io.Reader
	remaining int64
	err       error
	// countLines enables newlines and lastByte for blank-line rejection.
	countLines bool
	newlines   int
	lastByte   byte
	sawBytes   bool
}

// lines returns the physical lines delivered so far; a final line without a
// line break still counts.
func (reader *delimitedSourceLimitReader) lines() int {
	if reader.sawBytes && reader.lastByte != '\n' {
		return reader.newlines + 1
	}
	return reader.newlines
}

func (reader *delimitedSourceLimitReader) count(values []byte) {
	if !reader.countLines || len(values) == 0 {
		return
	}
	reader.newlines += bytes.Count(values, []byte{'\n'})
	reader.lastByte = values[len(values)-1]
	reader.sawBytes = true
}

func (reader *delimitedSourceLimitReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if reader.err != nil {
		return 0, reader.err
	}
	// One probe byte distinguishes an exact-length EOF from an over-budget source;
	// only the admitted prefix is returned and the excess becomes sticky.
	if reader.remaining < int64(len(p)) {
		p = p[:int(reader.remaining)+1]
	}
	n, err := reader.source.Read(p)
	if int64(n) > reader.remaining {
		n = int(reader.remaining)
		reader.remaining = 0
		reader.err = errDelimitedSourceLimit
		reader.count(p[:n])
		return n, reader.err
	}
	reader.remaining -= int64(n)
	reader.count(p[:n])
	return n, err
}

func (reader *delimitedRecordLimitReader) Read(destination []byte) (int, error) {
	if reader.limitErr != nil {
		return 0, reader.limitErr
	}
	count, err := reader.source.Read(destination)
	accepted, exceeded := reader.observeBytes(destination[:count])
	if exceeded {
		reader.limitErr = errDelimitedRecordLimit
		return accepted, reader.limitErr
	}
	return count, err
}

func (reader *delimitedRecordLimitReader) observeBytes(values []byte) (int, bool) {
	index := 0
	for index < len(values) {
		if reader.canObserveSimple() {
			consumed, exceeded := reader.observeSimple(values[index:])
			index += consumed
			if exceeded {
				return index, true
			}
			continue
		}
		if reader.canObserveQuotedRun() {
			consumed, exceeded := reader.observeQuotedRun(values[index:])
			index += consumed
			if exceeded {
				return index, true
			}
			continue
		}
		consumed, exceeded := reader.advance(1)
		index += consumed
		if exceeded {
			return index, true
		}
		reader.observe(values[index-1])
	}
	return index, false
}

func (reader *delimitedRecordLimitReader) canObserveQuotedRun() bool {
	switch {
	case !reader.inQuotes:
		return false
	case reader.quotePending:
		return false
	case reader.quoteCarriage:
		return false
	case reader.quoteDelimiterAt != 0:
		return false
	default:
		return true
	}
}

func (reader *delimitedRecordLimitReader) observeQuotedRun(values []byte) (int, bool) {
	quoteAt := bytes.IndexByte(values, '"')
	length := len(values)
	if quoteAt >= 0 {
		length = quoteAt + 1
	}
	consumed, exceeded := reader.advance(length)
	if !exceeded && quoteAt >= 0 {
		reader.quotePending = true
	}
	return consumed, exceeded
}

func (reader *delimitedRecordLimitReader) canObserveSimple() bool {
	return !reader.inQuotes && !reader.trimLeading && len(reader.delimiter) == 1 && len(reader.comment) == 0
}

func (reader *delimitedRecordLimitReader) observeSimple(values []byte) (int, bool) {
	quoteAt := bytes.IndexByte(values, '"')
	plainLength := len(values)
	if quoteAt >= 0 {
		plainLength = quoteAt
	}
	consumed, exceeded := reader.observePlain(values[:plainLength])
	if exceeded {
		return consumed, true
	}
	if quoteAt < 0 {
		return consumed, false
	}

	quoteConsumed, quoteExceeded := reader.advance(1)
	consumed += quoteConsumed
	if quoteExceeded {
		return consumed, true
	}
	if reader.atFieldStart {
		reader.inQuotes = true
	}
	reader.atFieldStart = false
	return consumed, false
}

func (reader *delimitedRecordLimitReader) observePlain(values []byte) (int, bool) {
	index := 0
	for index < len(values) {
		newlineAt := bytes.IndexByte(values[index:], '\n')
		if newlineAt < 0 {
			consumed, exceeded := reader.advance(len(values) - index)
			index += consumed
			if !exceeded {
				reader.atFieldStart = values[len(values)-1] == reader.delimiter[0]
			}
			return index, exceeded
		}

		length := newlineAt + 1
		consumed, exceeded := reader.advance(length)
		index += consumed
		if exceeded {
			return index, true
		}
		reader.resetRecord()
	}
	return index, false
}

func (reader *delimitedRecordLimitReader) advance(length int) (int, bool) {
	remaining := reader.maximum - reader.recordBytes
	if length > remaining {
		reader.recordBytes = reader.maximum
		return remaining, true
	}
	reader.recordBytes += length
	return length, false
}

func (reader *delimitedRecordLimitReader) observe(value byte) {
	if reader.inQuotes {
		reader.observeQuoted(value)
		return
	}
	if reader.inComment {
		if value == '\n' {
			reader.resetRecord()
		}
		return
	}
	if reader.lineStart && len(reader.comment) > 0 {
		if value == reader.comment[reader.commentAt] {
			reader.commentAt++
			if reader.commentAt == len(reader.comment) {
				reader.inComment = true
				reader.lineStart = false
			}
			return
		}
		reader.lineStart = false
		for _, prefix := range reader.comment[:reader.commentAt] {
			reader.observeRecordByte(prefix)
		}
		reader.commentAt = 0
	}
	reader.lineStart = false
	reader.observeRecordByte(value)
}

func (reader *delimitedRecordLimitReader) observeRecordByte(value byte) {
	if value == '\n' {
		reader.resetRecord()
		return
	}
	if reader.atFieldStart {
		if !reader.trimLeading {
			reader.observeUnquoted(value)
			return
		}
		reader.observeLeading(value)
		return
	}
	reader.observeUnquoted(value)
}

func (reader *delimitedRecordLimitReader) observeQuoted(value byte) {
	if reader.quoteCarriage {
		reader.quoteCarriage = false
		if value == '\n' {
			reader.inQuotes = false
			reader.quotePending = false
			reader.resetRecord()
			return
		}
		reader.quotePending = false
		reader.observeQuotedContent(value)
		return
	}
	if reader.quoteDelimiterAt > 0 {
		if value == reader.delimiter[reader.quoteDelimiterAt] {
			reader.quoteDelimiterAt++
			if reader.quoteDelimiterAt == len(reader.delimiter) {
				reader.inQuotes = false
				reader.quotePending = false
				reader.quoteDelimiterAt = 0
				reader.atFieldStart = true
				reader.leadingBytesAt = 0
			}
			return
		}
		reader.quoteDelimiterAt = 0
		reader.quotePending = false
		reader.observeQuotedContent(value)
		return
	}
	if !reader.quotePending {
		reader.observeQuotedContent(value)
		return
	}
	switch value {
	case '"':
		reader.quotePending = false
	case '\n':
		reader.inQuotes = false
		reader.quotePending = false
		reader.resetRecord()
	case '\r':
		reader.quoteCarriage = true
	default:
		if value == reader.delimiter[0] {
			reader.quoteDelimiterAt = 1
			if len(reader.delimiter) == 1 {
				reader.inQuotes = false
				reader.quotePending = false
				reader.quoteDelimiterAt = 0
				reader.atFieldStart = true
				reader.leadingBytesAt = 0
			}
			return
		}
		reader.quotePending = false
		reader.observeQuotedContent(value)
	}
}

func (reader *delimitedRecordLimitReader) observeQuotedContent(value byte) {
	if value == '"' {
		reader.quotePending = true
	}
}

func (reader *delimitedRecordLimitReader) observeLeading(value byte) {
	reader.leadingBytes[reader.leadingBytesAt] = value
	reader.leadingBytesAt++
	bytes := reader.leadingBytes[:reader.leadingBytesAt]
	if !utf8.FullRune(bytes) {
		return
	}

	runeValue, size := utf8.DecodeRune(bytes)
	switch {
	case runeValue == utf8.RuneError:
	case size != reader.leadingBytesAt:
	case unicode.IsSpace(runeValue):
		reader.leadingBytesAt = 0
		return
	}
	for _, buffered := range bytes {
		reader.observeUnquoted(buffered)
	}
	reader.leadingBytesAt = 0
}

func (reader *delimitedRecordLimitReader) observeUnquoted(value byte) {
	if value == '"' {
		if reader.delimiterAt != 0 {
			_ = reader.observeDelimiter(value)
		}
		if reader.atFieldStart {
			reader.inQuotes = true
			reader.quotePending = false
		}
		reader.atFieldStart = false
		reader.delimiterAt = 0
		return
	}
	if reader.observeDelimiter(value) {
		return
	}
	reader.atFieldStart = false
}

func (reader *delimitedRecordLimitReader) observeDelimiter(value byte) bool {
	if len(reader.delimiter) == 0 {
		return false
	}
	if value == reader.delimiter[reader.delimiterAt] {
		reader.delimiterAt++
		if reader.delimiterAt == len(reader.delimiter) {
			reader.delimiterAt = 0
			reader.atFieldStart = true
		}
		return true
	}
	reader.atFieldStart = false
	reader.delimiterAt = 0
	if value == reader.delimiter[0] {
		reader.delimiterAt = 1
		return true
	}
	return false
}

func (reader *delimitedRecordLimitReader) resetRecord() {
	reader.recordBytes = 0
	reader.inQuotes = false
	reader.quotePending = false
	reader.quoteCarriage = false
	reader.quoteDelimiterAt = 0
	reader.delimiterAt = 0
	reader.atFieldStart = true
	reader.leadingBytesAt = 0
	reader.lineStart = true
	reader.commentAt = 0
	reader.inComment = false
}

func validDelimiter(delimiter rune) bool {
	return delimiter != 0 && delimiter != '"' && delimiter != '\r' &&
		delimiter != '\n' && delimiter != utf8.RuneError
}

func cloneHeaderConfig(config *HeaderConfig) *HeaderConfig {
	if config == nil {
		return nil
	}
	cloned := *config
	if config.Replace != nil {
		cloned.Replace = make(map[string]string, len(config.Replace))
		for original, replacement := range config.Replace {
			cloned.Replace[original] = replacement
		}
	}
	return &cloned
}
