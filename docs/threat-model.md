# Tabular security threat model

Version: 2 (2026-10-02), describing the v2 source contract. Public tags and
releases establish actual publication and consumer adoption state.

Owner: tabular maintainers

## Scope and assets

The package protects process availability and integrity while converting
caller-supplied bytes into rows. Source confidentiality, returned row values,
stable error classification, and the absence of implicit filesystem or network
effects are maintained assets. Upload authentication, authorization, malware
scanning, durable storage, and business schema validation remain outside the
package boundary.

## Trust boundaries and mitigations

| Boundary | Attacker control | Maintained mitigation |
| --- | --- | --- |
| Delimited and fixed-width readers | bytes, delimiters, quotes, encodings, record and field lengths | finite raw-record admission before parsing; delimited parsed-field validation and fixed-width configured slicing; malformed input fails with typed errors |
| ZIP reader | central-directory metadata, names, methods, compressed and expanded bytes | finite source-size, entry-count, entry-size, total-size, and compression-ratio defaults; unsafe and duplicate names fail closed; CRC and declared sizes are verified while reading |
| XLS OLE/BIFF parser | sector graphs, directory metadata, record lengths, indexes, strings, and cells | 64 MiB source default; BIFF8 dimension and cumulative materialized-slot/sheet admission before dense/presence allocation; repeated/overlapping sheet offsets rejected; checked sector chains; no execution |
| XLSX/OOXML parser | ZIP/XML structure, relationships, sheets, sparse cells, formulas, and values | source and expansion limits precede Excelize; one unambiguous canonical workbook at `xl/workbook.xml`; local worksheet targets may be relocated; selected worksheet XML must be a complete document; malformed XML and external or traversing relationships fail closed; worksheet, row, and field defaults are finite |
| Normalization and errors | headers, replacements, cell values, parser diagnostics | no implicit formula execution; caller maps are copied; default Error diagnostics contain only fixed category/context and numeric coordinates, never cause text; explicit cause inspection is trusted |
| Dependencies and automation | Excelize, x/text, reusable workflow, analysis tools | reviewed module pins, checksums, immutable workflow pinning, vulnerability and static-security gates, and release-time dependency review |

The package performs no network, process, environment, database, queue, or
cache access. It never writes archive entries to disk, follows archive links,
starts goroutines, retries work, or evaluates formulas. Stateful readers are
caller-serialized and caller-provided sources retain ownership.

## Abuse cases

- Oversized records, fields, workbooks, worksheet counts, archive sources or expansions,
  or compression ratios fail with `ErrorLimitExceeded` under finite defaults.
- ZIP traversal, absolute or backslash paths, duplicate names, corrupt sizes,
  CRC failures, unsupported methods, and malformed OOXML fail before caller use.
- OLE cycles, out-of-range sectors, invalid stream sizes, truncated BIFF
  records, and invalid cell references fail without recovery heuristics.
- External OOXML worksheet relationships fail without network access. Formula
  text is data; callers exporting values to formula-aware systems must apply
  their own output-encoding policy.

## Accepted residual risks

| Risk | Severity | Rationale and mitigation | Review condition |
| --- | --- | --- | --- |
| ZIP directory, sheet XML, Excelize and legacy XLS materialization can allocate more heap than input bytes within accepted limits. | Medium | Maintainers own the risk. ZIP entries and XML sheet counts are checked after index/list materialization; spreadsheet row/field limits validate decoded values, not parser allocation. Source and expansion byte defaults bound input, not heap. Callers select lower limits and job-level memory controls. | Review when dependencies, defaults, or materialization change, or amplification exceeds deployment budgets. |
| Synchronous parsing and caller I/O cannot be interrupted by the package. | Medium | Maintainers own the API compatibility tradeoff; integrators own source lifetime, cancellable/deadline-aware Reader/ReaderAt/Writer implementations and job isolation. Delimited total-source admission includes skipped bytes, but byte/slot limits do not establish an elapsed-time guarantee for caller callbacks or parser delegates. | Review when cancellation APIs or dependencies change, or deployment input exceeds job budgets. |
| Trusted normalization/configuration and explicit cause inspection can expand output or expose private values. | Low | Maintainers preserve caller-selected EmptyAs/Replace values and original causes; integrators bound configuration/output and restrict Err/Unwrap/Is/As introspection to protected handling. Default diagnostics do not traverse or invoke causes. | Review when configuration trust, normalization order, or diagnostic APIs change. |
| ZIP symlink metadata is accepted by default. | Low | Entries are never followed or written to a filesystem; callers that prohibit link metadata can set `RejectSymlinks`. | Review if extraction-to-path behavior is ever proposed. |

No Critical or High residual source finding is accepted by this model. Any new
parser, format, implicit I/O, background work, or security-sensitive diagnostic
must update this model and add hostile-input evidence before release.
