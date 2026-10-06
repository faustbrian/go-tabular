# Quickstart

Install the published v2 module:

```sh
go get github.com/faustbrian/go-tabular/v2@v2.0.0
```

The guide describes the public v2 contract, including fields absent in v1.
Import the `/v2` path; do not combine v1 installation with the v2-only limits
shown below. Existing v1 consumers should follow the [migration guide](migration.md).

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
go test github.com/faustbrian/go-tabular/v2 -run '^ExampleNewDelimitedReader$' -v
```

Its complete source is
[`ExampleNewDelimitedReader`](../example_test.go). It constructs a bounded
semicolon reader, validates a normalized header, reads all rows, and handles
both `io.EOF` and non-terminal failures. The package-level examples remain the
executable authority for snippets in this guide.

## Configure limits

Defaults are protective, not unlimited. Set limits from the surrounding
system's upload policy when those limits are smaller. XLS and XLSX use
`MaxWorkbookBytes`; XLSX additionally uses the limits in `SpreadsheetConfig.ZIP`.
Untrusted spreadsheets should also set `MaxRecordBytes` and `MaxFieldBytes`
to bound parsed rows and cells before normalization and caller delivery. Set
`MaxSourceBytes` on delimited readers to bound total bytes, including comments
and blank lines. XLS `MaxMaterializedCells` bounds cumulative dense slots before
materialization, while `MaxSheets` bounds both workbook formats. For
XLSX, override `MaxSheets`, `ZIP.MaxArchiveBytes`, and
`ZIP.MaxCompressionRatio` when the accepted
workbook policy differs, and set `ZIP.RejectSymlinks` if metadata links are not
allowed.

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
remain available after construction. XLSX captures a bounded owned revision
before validation; parsing and presence-aware iteration use that revision, so
its source also need not remain available after successful construction.

All readers are stateful and require caller serialization. The package starts
no goroutines and has no shutdown sequence beyond the documented `Close`
methods. Parsing uses the frozen context-free codec exception: reads are
synchronous, so cancellation and deadlines must be supplied by the source or
the calling goroutine rather than a package context parameter.
