// Package tabular provides explicit, bounded readers for tabular ingest.
//
// CSV and configurable delimited input, fixed-width records, XLS, XLSX, and
// ZIP-backed sources share deterministic rows and typed errors. Parsers do not
// auto-detect formats or silently apply normalization. Callers choose the
// format, limits, header rules, and field transformations in configuration.
//
// Delimited, fixed-width, ZIP entry, and XLSX row processing stream input.
// Legacy XLS workbooks are materialized up to MaxWorkbookBytes because the
// OLE2/BIFF8 format requires random access to workbook structures.
//
// The package never closes caller-provided sources. Delimited and fixed-width
// readers retain their io.Reader, and ZIPArchive retains its io.ReaderAt.
// OpenSpreadsheet consumes XLS input during construction; XLSX callers should
// keep the source available until Close because presence-aware iteration
// retains archive entry readers. Spreadsheet readers and readers returned by
// ZIPArchive.Open own internal resources that callers must close. The package
// starts no goroutines, performs no retries, and has no hidden shutdown phase.
// Stateful reader values are not safe for concurrent method calls.
//
// Parsing intentionally has no context parameter: operations synchronously
// consume the supplied io.Reader or io.ReaderAt and perform no network I/O.
// Callers provide cancellation or deadline behavior through those sources and
// their calling goroutines.
package tabular
