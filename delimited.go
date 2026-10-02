package tabular

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"unicode"
	"unicode/utf8"
)

const delimitedReadBufferSize = 64 * 1024

var errDelimitedRecordLimit = errors.New("delimited record exceeds byte limit")
var errDelimitedSourceLimit = errors.New("delimited source exceeds byte limit")

const defaultMaxDelimitedSourceBytes int64 = 64 * 1024 * 1024

// DelimitedConfig explicitly controls delimited-text parsing behavior.
type DelimitedConfig struct {
	Delimiter           rune
	Comment             rune
	LazyQuotes          bool
	TrimLeadingSpace    bool
	AllowVariableFields bool
	FieldsPerRecord     int
	// MaxRecordBytes bounds one logical record before parser allocation.
	// Zero selects a finite 1 MiB default.
	MaxRecordBytes int
	// MaxFieldBytes bounds one parsed field before normalization.
	// Zero selects a finite 1 MiB default.
	MaxFieldBytes int
	// MaxSourceBytes bounds all bytes consumed, including headers, blank lines and
	// comments, before CSV parsing. Zero selects 64 MiB; positive values override it.
	MaxSourceBytes int64
	Header         *HeaderConfig
	Normalize      NormalizationConfig
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
}

// NewCSVReader constructs a comma-delimited streaming reader.
func NewCSVReader(source io.Reader, config DelimitedConfig) (*DelimitedReader, error) {
	config.Delimiter = ','
	reader, err := newDelimitedReader(source, config, "csv")
	if err != nil {
		return nil, err
	}
	return reader, nil
}

// NewDelimitedReader constructs a streaming reader for an explicit delimiter.
func NewDelimitedReader(source io.Reader, config DelimitedConfig) (*DelimitedReader, error) {
	return newDelimitedReader(source, config, "delimited")
}

func newDelimitedReader(source io.Reader, config DelimitedConfig, format string) (*DelimitedReader, error) {
	if source == nil || !validDelimiter(config.Delimiter) ||
		(config.Comment != 0 && (!validDelimiter(config.Comment) || config.Comment == config.Delimiter)) ||
		config.FieldsPerRecord < 0 || config.MaxRecordBytes < 0 ||
		config.MaxFieldBytes < 0 || config.MaxSourceBytes < 0 {
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
	parserSource := &delimitedRecordLimitReader{
		source:       &delimitedSourceLimitReader{source: source, remaining: maxSourceBytes},
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
	}, nil
}

// Header returns the normalized first row when header handling is configured.
// The returned row is a copy and is safe for the caller to modify.
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

// Read returns the next normalized record. io.EOF marks a clean end of input.
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
	record, err := reader.reader.Read()
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
}

func (reader *delimitedSourceLimitReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if reader.err != nil {
		return 0, reader.err
	}
	if reader.remaining < int64(len(p)) {
		p = p[:int(reader.remaining)+1]
	}
	n, err := reader.source.Read(p)
	if int64(n) > reader.remaining {
		n = int(reader.remaining)
		reader.remaining = 0
		reader.err = errDelimitedSourceLimit
		return n, reader.err
	}
	reader.remaining -= int64(n)
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
