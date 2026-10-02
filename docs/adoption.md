# Adoption guide

## V2 migration

This source tree uses `github.com/faustbrian/go-tabular/v2`. Public tags and
releases establish availability after release gates pass. The owned consumer
boundaries requiring deliberate adoption are:

- `go-filesystem/integration/tabular-ingestion`;
- `go-library-tools/release/compatibility-consumer`.

After v2 publication, their owners must update imports deliberately and verify
the affected composition or release-compatibility boundary. Existing v1
consumers retain published v1 behavior and must configure explicit positive
limits for untrusted input; finite zero-value defaults are the v2 source contract.

1. Identify the source format and encoding from its contract. Do not infer
   either from filenames alone.
2. Set file, archive, and record limits from the narrowest upstream policy.
3. Decide whether the first row is a header and make duplicate/empty handling
   explicit.
4. Decide row-width policy before processing data.
5. Keep normalization minimal and record every data-changing option.
6. Match `ErrorKind` values for operational decisions. Read coordinates from
   `Error`. Default `Error.Error()` messages omit cause text and unknown metadata;
   explicit cause inspection through `Err`, `Unwrap`, or `errors.As` is trusted
   handling and can expose parser details or caller-provided entry names.
7. Drain or close readers on every path. Keep delimited, fixed-width, ZIP, and
   XLSX sources available while their readers consume them. XLS is fully
   materialized during `OpenSpreadsheet`, so its source may close after that
   call succeeds.

For ZIP-backed imports, call `OpenZIP`, select an exact entry from `Entries`,
and pass the opened entry to a streaming text reader. XLS/XLSX require
`io.ReaderAt`; copy only the selected ZIP member into a bounded random-access
store if spreadsheet files are archived.

Roll out against captured, sanitized production fixtures. Compare row counts,
headers, empty values, numeric strings, error classes, and rejected records
with the existing importer before replacing it.

Do not share one reader instance across goroutines. If cancellation is needed,
use a source whose `Read` can be interrupted and stop consuming it from the
owning goroutine; the codec API intentionally has no context parameter.
