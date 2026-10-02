package tabular

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorKind identifies a stable class of error that callers can match with
// errors.Is.
type ErrorKind string

const (
	// ErrorInvalidConfig indicates invalid parser configuration.
	ErrorInvalidConfig ErrorKind = "invalid configuration"
	// ErrorInvalidHeader indicates a missing or invalid header.
	ErrorInvalidHeader ErrorKind = "invalid header"
	// ErrorDuplicateHeader indicates a duplicate normalized header name.
	ErrorDuplicateHeader ErrorKind = "duplicate header"
	// ErrorMalformedRow indicates invalid record syntax or shape.
	ErrorMalformedRow ErrorKind = "malformed row"
	// ErrorInvalidEncoding indicates unsupported or invalid text encoding.
	ErrorInvalidEncoding ErrorKind = "invalid encoding"
	// ErrorInvalidLayout indicates an invalid fixed-width layout.
	ErrorInvalidLayout ErrorKind = "invalid fixed-width layout"
	// ErrorArchive indicates an invalid archive or entry.
	ErrorArchive ErrorKind = "archive error"
	// ErrorEntryNotFound indicates an absent exact archive entry name.
	ErrorEntryNotFound ErrorKind = "archive entry not found"
	// ErrorLimitExceeded indicates a configured resource limit was exceeded.
	ErrorLimitExceeded ErrorKind = "limit exceeded"
	// ErrorSpreadsheet indicates an invalid workbook or worksheet operation.
	ErrorSpreadsheet ErrorKind = "spreadsheet error"
)

func (kind ErrorKind) Error() string {
	return string(kind)
}

// Error carries stable classification and optional ingest coordinates.
// Row and Field are one-based when set. Default diagnostics omit cause text and
// unknown metadata. Err and Unwrap retain the original cause for trusted inspection.
type Error struct {
	Kind   ErrorKind
	Op     string
	Format string
	Row    int
	Field  int
	Err    error
}

func (err *Error) Error() string {
	var context strings.Builder
	context.WriteString("tabular")
	if knownDiagnosticContext(err.Op) {
		context.WriteString(": ")
		context.WriteString(err.Op)
	}
	if knownDiagnosticContext(err.Format) {
		context.WriteByte(' ')
		context.WriteString(err.Format)
	}
	if err.Row > 0 {
		fmt.Fprintf(&context, " row %d", err.Row)
	}
	if err.Field > 0 {
		fmt.Fprintf(&context, " field %d", err.Field)
	}
	context.WriteString(": ")
	switch err.Kind {
	case ErrorInvalidConfig, ErrorInvalidHeader, ErrorDuplicateHeader,
		ErrorMalformedRow, ErrorInvalidEncoding, ErrorInvalidLayout,
		ErrorArchive, ErrorEntryNotFound, ErrorLimitExceeded, ErrorSpreadsheet:
		context.WriteString(string(err.Kind))
	default:
		context.WriteString("error")
	}
	return context.String()
}

// Only package-owned literal context is safe in the default diagnostic. Public
// metadata and unsupported format/encoding values may contain caller data.
func knownDiagnosticContext(value string) bool {
	switch value {
	case "delimited.new", "delimited.header", "delimited.read",
		"fixedwidth.new", "fixedwidth.read", "fixedwidth.extract",
		"zip.open", "zip.entry.open", "zip.entry.read", "zip.extract",
		"encoding.decode", "encoding.reader", "encoding.decoder", "header.normalize",
		"spreadsheet.open", "spreadsheet.sheet", "spreadsheet.rows",
		"spreadsheet.read", "spreadsheet.header", "spreadsheet.validate", "spreadsheet.presence",
		"csv", "delimited", "fixed-width", "zip", "xls", "xlsx",
		"utf-8", "iso-8859-1", "windows-1252":
		return true
	default:
		return false
	}
}

func (err *Error) Unwrap() error {
	return err.Err
}

// Is matches stable ErrorKind values and wrapped causes.
func (err *Error) Is(target error) bool {
	kind, ok := target.(ErrorKind)
	if ok && err.Kind == kind {
		return true
	}
	return errors.Is(err.Err, target)
}
