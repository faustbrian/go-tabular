# Public API reference

The authoritative signature documentation is available through
`go doc github.com/faustbrian/go-tabular`. This page groups the surface by
task and records compatibility-sensitive semantics.

The repository has one public package and no public subpackages, adapters, or
test-helper packages. The `internal/xls` implementation cannot be imported by
consumers.

## Rows and normalization

- `Row []string`: ordered source fields.
- `NormalizationConfig`: optional whitespace trimming and empty replacement.
- `NormalizeRow`: returns a copy.
- `HeaderConfig`, `HeaderCase`, and `NormalizeHeader`: BOM removal, trimming,
  case conversion, replacements, and empty/duplicate validation.

## Delimited text

- `DelimitedConfig`: delimiter, comments, quote policy, row shape, header,
  normalization, and explicit logical-record and field byte limits.
- Zero record and field limits preserve unbounded legacy behavior; untrusted
  sources require explicit positive limits.
- `NewCSVReader`: always selects comma, regardless of `Delimiter`.
- `NewDelimitedReader`: requires a valid explicit delimiter.
- `DelimitedReader.Header` and `Read`: cached header and streaming rows.

## Fixed-width text

- `FixedWidthField`: named half-open byte interval `[Start, End)`.
- `FixedWidthConfig`: layout, source encoding, short/trailing record policy,
  maximum record size, and normalization.
- `NewFixedWidthReader`, `Fields`, and `Read`: validated streaming parser.
- `ExtractBytes`: non-copying checked byte slice.

## Encodings

- `EncodingUTF8`, `EncodingISO88591`, and `EncodingWindows1252` are supported.
- `DecodeBytes` validates/converts a complete value.
- `DecodeReader` returns a validating/converting streaming reader.

## Archives

- `ZIPConfig`: maximum entries, per-entry bytes, total expanded bytes,
  compression ratio, and symbolic-link rejection.
- `OpenZIP`: validates and indexes a random-access ZIP source.
- `ZIPArchive.Entries`, `Open`, and `Extract`: copied metadata, exact entry
  streaming, and writer-based extraction.

## Spreadsheets

- `FormatXLS` and `FormatXLSX` must be selected explicitly.
- `SpreadsheetConfig`: sheet, headers, row shape, errors, workbook, worksheet
  count and XLSX ZIP limits, plus opt-in parsed-row, cell byte, and cell
  presence behavior.
- Zero spreadsheet record and field limits preserve legacy behavior;
  untrusted sources require explicit positive limits.
- `OpenSpreadsheet`, `Header`, `Read`, and `Close`: common string-row
  lifecycle.
- `PreserveCellPresence`, `SpreadsheetCell`, `SpreadsheetRow`, and
  `ReadCells`: opt-in distinction between absent and stored-empty cells.

## Errors

`Error` includes `Kind`, operation, format, one-based row/field coordinates,
and a wrapped cause. Match stable categories with `errors.Is(err,
ErrorMalformedRow)` and inspect details with `errors.As`.

Stable kinds are `ErrorInvalidConfig`, `ErrorInvalidHeader`,
`ErrorDuplicateHeader`, `ErrorMalformedRow`, `ErrorInvalidEncoding`,
`ErrorInvalidLayout`, `ErrorArchive`, `ErrorEntryNotFound`,
`ErrorLimitExceeded`, and `ErrorSpreadsheet`.

Wrapped causes can contain source-controlled parser details or caller-provided
archive names. Match `ErrorKind` for control flow and sanitize complete error
strings before placing them in logs, traces, or responses.

## Lifecycle And Concurrency

- Delimited and fixed-width readers borrow their `io.Reader`; they have no
  `Close` method and do not close the source.
- `OpenZIP` borrows its `io.ReaderAt`. Callers close every successful
  `ZIPArchive.Open` result; `Extract` closes its own temporary entry reader.
- `OpenSpreadsheet` never closes its `io.ReaderAt`. XLS is fully consumed by
  the open call. Keep XLSX sources available until `SpreadsheetReader.Close`
  because presence-aware iteration retains archive entry readers. Closing the
  spreadsheet reader releases only package-owned iterator resources.
- Reader methods mutate cursor/header state and are not safe for concurrent
  calls. The package starts no goroutines and exposes no channels or callbacks.
- Parsing is synchronous and intentionally context-free. Cancellation and
  deadlines must be implemented by the source or caller.
