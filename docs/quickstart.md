# Quickstart

Install the stable root module:

```sh
go get github.com/faustbrian/go-tabular@v1
```

## Choose a reader

Use `NewCSVReader` for comma-separated data, `NewDelimitedReader` for an
explicit non-comma delimiter, `NewFixedWidthReader` for byte-positioned
records, and `OpenSpreadsheet` for an explicitly selected XLS or XLSX file.
Use `OpenZIP` when an import arrives inside an archive.

Every streaming reader returns `io.EOF` after its last row. Treat any other
error as a failed import unless the application has an explicit recovery
policy.

Run the compiler-checked delimited quick start:

```sh
go test github.com/faustbrian/go-tabular -run '^ExampleNewDelimitedReader$' -v
```

Its complete source is
[`ExampleNewDelimitedReader`](../example_test.go). It constructs a bounded
semicolon reader, validates a normalized header, reads all rows, and handles
both `io.EOF` and non-terminal failures. The package-level examples remain the
executable authority for snippets in this guide.

## Configure limits

Defaults are protective, not unlimited. Set limits from the surrounding
system's upload policy when those limits are smaller. XLS uses
`MaxWorkbookBytes`; XLSX uses the limits in `SpreadsheetConfig.ZIP`.
Untrusted spreadsheets should also set `MaxRecordBytes` and `MaxFieldBytes`
to bound parsed rows and cells before normalization and caller delivery. For
XLSX, set `MaxSheets`, `ZIP.MaxCompressionRatio`, and `ZIP.RejectSymlinks` from
the accepted workbook policy.

When an import distinguishes missing cells from explicitly stored empty
strings, set `PreserveCellPresence` and call `ReadCells`. The default `Read`
method intentionally returns only string values and preserves its existing
optimized behavior.

## Handle typed errors

Match stable categories with `errors.Is` and inspect coordinates with
`errors.As`. Treat the wrapped cause and any archive entry name as untrusted;
sanitize them before logging. The compiler-checked examples fail closed on any
non-`io.EOF` error.

## Close spreadsheets and ZIP entries

Call `Close` on spreadsheet readers and ZIP entry readers. Closing a
spreadsheet does not close the caller-owned `io.ReaderAt`.

Delimited and fixed-width readers do not own or close their input. `OpenZIP`
retains the caller's `io.ReaderAt`, while `ZIPArchive.Open` returns a new
caller-closed entry reader. `ZIPArchive.Extract` opens and closes its own entry
reader. XLS input is fully consumed during `OpenSpreadsheet`, so it need not
remain available after construction. Keep an XLSX source available until the
spreadsheet reader closes because presence-aware iteration retains archive
entry readers.

All readers are stateful and require caller serialization. The package starts
no goroutines and has no shutdown sequence beyond the documented `Close`
methods. Parsing uses the frozen context-free codec exception: reads are
synchronous, so cancellation and deadlines must be supplied by the source or
the calling goroutine rather than a package context parameter.
