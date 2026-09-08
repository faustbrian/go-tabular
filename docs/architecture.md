# Architecture

The public package owns configuration, stable errors, normalization, and the
common row contract. Each parser translates its format into ordered `Row`
values without schema inference.

Delimited input delegates RFC-style quoting and record boundaries to Go's
`encoding/csv`, then adds explicit header and normalization semantics.
Fixed-width input scans bounded newline records and slices byte ranges before
decoding. ZIP input validates the central directory before exposing exact
entry readers.

XLSX first validates ZIP limits and worksheet XML well-formedness, then uses
Excelize's row iterator with raw cell values. When cell presence is enabled, a
second streaming decoder follows the selected worksheet's actual `<c>`
elements so numeric cells and stored-empty trailing cells remain distinguishable
from absent positions. XLS is intentionally separate under `internal/xls`; it
reads OLE2 compound streams and a limited BIFF8 record set into bounded memory.

This split keeps a single public spreadsheet API while preserving honest
format-specific resource behavior. Internal XLS types are not compatibility
surface. Excelize is isolated behind narrow iterator interfaces so its errors
remain testable and callers do not depend on its types.

## Ownership And Lifecycle

The root `tabular` package is the only public package. It owns decoding policy,
normalization, configured limits, row projection, and stable error categories.
The application owns source acquisition, transport limits, schema conversion,
persistence, orchestration, and closing each supplied `io.Reader` or
`io.ReaderAt`.

Constructors copy retained mutable configuration such as fixed-width field
definitions and header replacement maps. Returned rows and archive entry lists
do not expose package-owned mutable state. Delimited and fixed-width readers
borrow and retain their sources but hold no closeable package resource.
`ZIPArchive` retains its random-access source. XLS is fully materialized during
`OpenSpreadsheet`, while XLSX callers should retain the source until close
because presence-aware iteration keeps archive entry readers. A spreadsheet
reader owns its internal iterator and must be closed; closing it does not close
the source. Each reader returned by `ZIPArchive.Open` is caller-owned and must
be closed. `ZIPArchive.Extract` owns and closes its temporary entry reader.

## Cancellation And Concurrency

The parsers use the frozen context-free codec exception. They synchronously
consume caller-provided in-memory or streaming sources, perform no network I/O,
start no goroutines, schedule no retries, and expose no callbacks or channels.
Cancellation and deadlines therefore belong to the source implementation and
the calling goroutine.

Reader instances contain cursor and header state and are not safe for
concurrent method calls. Independent readers may be used concurrently when
their underlying sources permit it. Configuration values can be reused after
construction because retained mutable fields are copied.
