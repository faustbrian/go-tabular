# Performance and memory verification

Benchmarks generate representative inputs before timing so the repository does
not carry large derived fixtures. `go test ./... -run '^$' -bench . -benchmem`
exercises:

- 20,000 CSV rows with three fields;
- 20,000 fixed-width rows with three byte-positioned fields;
- ZIP extraction of a 20,000-row CSV member;
- an 8 MiB bounded XLS source; and
- a 10,000-row XLSX workbook written with a streaming OOXML writer.

Every benchmark reports allocations and bytes processed. Results are runtime,
architecture, Go-version, and dependency-version specific; the project does
not publish universal throughput or heap guarantees from a single machine.

Delimited and fixed-width constructors do not read input, and regression tests
feed one-byte chunks to prove the first row can be returned without consuming
the complete source. ZIP entry extraction is streamed through `io.Copy`.

XLS necessarily materializes its source and rejects files above
`MaxWorkbookBytes`. XLSX returns rows incrementally, but validation and
Excelize may allocate substantially more heap than the compressed workbook
size. ZIP limits bound declared expanded payload sizes; callers must also set
process/job memory limits appropriate to their environment.
