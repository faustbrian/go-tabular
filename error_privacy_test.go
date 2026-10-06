package tabular

import (
	"errors"
	"strings"
	"testing"
)

type diagnosticSourceError struct{}

func (*diagnosticSourceError) Error() string { return "ordinary-private-marker" }

type diagnosticReader struct{ err error }

func (reader diagnosticReader) Read([]byte) (int, error) { return 0, reader.err }

func TestDefaultDiagnosticPreservesTrustedCauseWithoutDisclosure(t *testing.T) {
	cause := &diagnosticSourceError{}
	reader, err := NewCSVReader(diagnosticReader{err: cause}, DelimitedConfig{})
	if err != nil {
		t.Fatal(err)
	}
	row, err := reader.Read()
	if row != nil || !errors.Is(err, ErrorMalformedRow) || !errors.Is(err, cause) {
		t.Fatalf("Read classification/atomicity = %v, %v", row, err)
	}
	if got := err.Error(); got != "tabular: delimited.read csv row 1: malformed row" {
		t.Fatalf("default diagnostic = %q", got)
	}
	if errors.Unwrap(err) != cause { //nolint:errorlint // Require direct cause identity, not wrapped equivalence.
		t.Fatal("direct cause identity changed")
	}
	var typed *diagnosticSourceError
	if !errors.As(err, &typed) || typed != cause {
		t.Fatal("typed cause identity changed")
	}
}

func TestDefaultDiagnosticDoesNotDiscloseRequestedEntryName(t *testing.T) {
	archive := &ZIPArchive{}
	_, err := archive.Open("ordinary-private-marker")
	if !errors.Is(err, ErrorEntryNotFound) || err.Error() != "tabular: zip.entry.open zip: archive entry not found" {
		t.Fatalf("missing entry = %v", err)
	}
	if cause := errors.Unwrap(err); cause == nil || cause.Error() != "ordinary-private-marker" {
		t.Fatal("trusted entry-name cause missing")
	}
}

func TestDefaultDiagnosticBoundsCallerMetadata(t *testing.T) {
	marker := "ordinary-private-marker"
	_, err := DecodeBytes(nil, Encoding(marker))
	if !errors.Is(err, ErrorInvalidEncoding) || strings.Contains(err.Error(), marker) {
		t.Fatalf("unsupported encoding diagnostic = %v", err)
	}
	err = &Error{Kind: ErrorKind(marker), Op: marker, Format: marker, Row: 1, Field: 2, Err: &diagnosticSourceError{}}
	if got := err.Error(); got != "tabular row 1 field 2: error" {
		t.Fatalf("unknown metadata = %q", got)
	}
}
