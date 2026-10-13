# Source documentation audit

[Documentation index](README.md) · [Repository entry](../README.md)

## Scope and reproduction

Documentation-only audit of exact remote-main base
`0e1df19ea21067da124530ab547de1a8fc578685`. One module,
`github.com/faustbrian/go-tabular/v2`, owns public tabular and internal/xls.
The unrelated original checkout is not the source for this pass. Reproduce
inventory with `git ls-files '*.go'`, modules.json, packages.json and
`golib source-docs inventory --json`. Every tracked production, internal,
test, fuzz, benchmark, example and platform/build-constrained file is included.

Before comment edits, structural inventory reported 57 undocumented public/
internal-exported fields and methods. Both canonical package comments exist;
root ownership prose is stale for XLSX's now-owned source snapshot. Semantic
investigation covers source versus snapshot ownership, exact parser admission
measures, EOF/partial-error behavior, header caching, row/presence interpretation,
non-contextual synchronous I/O, hostile ZIP/XML/CFB/BIFF inputs, numeric/raw
values, delegates and deliberately incomplete format support. Add contracts
and invariant rationale, not generic comments on every private fixture member.

Tier A owner is this repository. Direct consumers are ingestion code using
Row/readers, archive entry readers and SpreadsheetRow presence, plus the root
package's internal XLS adapter. Preserve executable behavior and historical
spec/release evidence. Selected checks: format; native AST/non-comment tokens/
compiler-security directives against base; render both packages; native docs
(examples and vet); current source-doc inventory; local links, pinned fixture
bytes, changed-current-file secrets and ordinary complete final diff. API mode
is apidiff at api/v2.txt, not full-prose go-doc; neither native script nor
.golib.yaml requests a prose snapshot. Baselines/signatures remain unchanged.
No unrelated runtime/race/fuzz/large-benchmark/conformance/386/release campaign.

## Source-backed rationale and boundaries

- Text readers retain but never close caller io.Reader; OpenZIP retains its
  io.ReaderAt. OpenSpreadsheet consumes XLS during construction and snapshots
  size-bounded XLSX bytes before ZIP/XML admission, delegate and presence reads.
  Successful spreadsheet construction no longer needs the original source.
  It owns iterator/workbook resources requiring Close; Close marks closed even
  if cleanup returns an error, so later calls do not retry cleanup. Header is
  cached and has no independent closed-state check; Read/ReadCells refuse closed
  readers. Stateful readers must be externally serialized; no hidden goroutines,
  retries or cancellation mechanism are created by production code.
- Headers are consumed once only when configured; errors are cached. Config
  header replacement maps and fixed-width layouts are copied at construction.
  Returned rows/header/field/entry slices may be changed without changing parser
  state. ExtractBytes intentionally aliases the source slice, retaining its
  backing storage; no empty range is admitted. NormalizeRow/Header allocate new
  slices but share immutable strings. Standalone NormalizeHeader borrows its
  replacement map for the call and applies BOM/trim/case/replacement/validation
  in that order; replacement output is not normalized again.
- Delimited syntax follows encoding/csv, not strict RFC 4180 conformance. No
  character encoding is automatically decoded/validated by that parser; use
  DecodeReader explicitly. Record admission counts presented bytes including
  framing, quoted multiline data, comments and blank-line boundaries. Field
  limits count parsed string bytes before normalization. Total source admission
  counts skipped bytes too, reads one excess byte to distinguish exact EOF and
  has sticky refusal. Buffering may read ahead; prior complete rows are not
  rolled back. Partial CSV rows on errors are discarded by this API.
- Fixed-width offsets select original encoded bytes before decoding, allow
  gaps, require ordered nonoverlap and can split UTF-8 (which then fails).
  Short-record mode clips each field; trailing bytes may be ignored explicitly.
  MaxRecordBytes is a Scanner buffer/token ceiling, not an inclusive guarantee
  for payload length independently of line framing. All Scanner errors are
  currently ErrorLimitExceeded with their original cause retained.
- Spreadsheet field/record limits are checked after decoding and before
  normalization, counting output/error strings, not parser allocations. Width
  policy pads short rows with absent cells and rejects long rows; variable mode
  disables this. Unlike CSV, no-header/zero-width spreadsheets do not infer the
  first row width. Read and ReadCells consume the same cursor. ReadCells requires
  presence opt-in; XLS supported records and XLSX c elements determine presence,
  not a completeness guarantee for unsupported BIFF records. No formula execution,
  formatting/date inference, schema validation or full BIFF/OOXML support is owned.
- XLS admission charges dense worksheet slots across all sheets, at least one
  per absent/empty row, before materialization. Wire counts widen to uint64
  before host-int admission; physical sectors and BIFF8 dimensions bound
  conversion/indexing. Workbook/Book stream selection, cyclic chains, worksheet
  span overlap, SST continuation width and record payload checks are invariants,
  not permission to change ignored-record behavior. Numeric NUMBER/RK values are
  rendered from float64 without preserving Excel styles or original lexical text.
- ZIP source bounds precede indexing; count and expansion metadata limits follow
  central-directory materialization. Ratio comparison uses quotient/remainder
  without overflow. Safe names are exact canonical slash-relative paths, not
  filesystem destinations or sanitized filenames. Symlinks are never followed;
  optional rejection is metadata policy. CRC verification requires complete
  entry reads and is integrity, not authentication. Extract can write a partial
  destination before a failure and ignores Close errors; it has no rollback.
- XLSX admission, Excelize and presence use the same owned compressed revision.
  External/traversing/ambiguous relationships and unsupported canonical graphs
  fail before delegation. Strict-namespace text substitutions in the pinned
  Excelize dictionary must not transform sheet names/IDs/targets after admission;
  valid namespace vocabulary itself remains supported. Custom process-registered
  ZIP codecs are trusted, potentially repeated callbacks: a byte snapshot does
  not make their decoded output deterministic or cancel their work.
- Default Error.Error diagnostics allow only package-owned operation/format/kind
  vocabulary and positive coordinates. Err/Unwrap still expose the original
  parser/source/writer cause, possibly input metadata; raw struct inspection or
  generic encoding is not a redaction guarantee. ErrorKind values support
  errors.Is; custom ErrorKind.Error returns its raw string.
- Limits bound named input/accounting measures, not process heap/time. ZIP indexes,
  XML lists, XLS materialization and Excelize may amplify finite input. No context
  is accepted; callers own interruptible I/O/job isolation, including writers.

Evidence: every tracked source/assertion below, current ownership/admission
contracts in README and docs, and resolved TABULAR-DEC-001 through -008. Authority
identities are recorded in specification/manifest.json: RFC 4180; MS-CFB 12.0;
MS-XLS 12.2; MS-OI29500 25.0; ECMA-376 fifth edition; XML 1.0 fifth edition;
PKWARE APPNOTE 6.3.10. Go encoding/csv and the exact Excelize pseudo-version are
delegates, not independent full conformance oracles. No normative history is
rewritten and no online authority refresh is claimed.

## Variants, fixtures and assurance

No tracked generated Go source or go:generate operation exists. Four pinned
fixture hashes match the committed manifest; base64 companions and fixture
origin/license records remain unchanged. Test-only helpers construct synthetic
BIFF/ZIP/XML packages in memory; the largebench producer creates disposable
CSV/XLSX files under b.TempDir, not distributed generated source.

Three constrained files: large_benchmark_test.go uses largebench; two Unix
mutation-resource guards use an OS expression. Those guards only activate for
GOLIB_GREMLINS_COVERAGE_PROFILE, serializing profile access and enforcing heap/
wall ceilings with an explicit stop signal. Ordinary examples do not activate
that campaign. Directives and existing narrow security suppressions stay exact.
Six root fuzz targets cover delimited parsing, extraction, fixed-width, ZIP,
XLS and XLSX; comments state actual byte/config boundaries and panic/error/
traversal oracles, not full validity/conformance. Six ordinary benchmarks plus
two largebench tracks have different construction/setup and compressed-versus-
expanded byte measures; no speed or fixed-memory guarantee is inferred. Four
examples cover delimited, fixed-width, ZIP and spreadsheet ownership workflows.
No saved active-fuzz crash corpus or unfinished implementation marker was found.

## Tracked Go coverage

| File | Audit boundary |
| --- | --- |
| `admission_repair_test.go` | Finite parser admission and repaired boundary regressions |
| `archive.go` | ZIP admission, exact names, ratio arithmetic, borrowed source and partial writes |
| `archive_test.go` | ZIP safety, budgets, CRC/writer errors, fuzz traversal and extraction benchmark |
| `automation_test.go` | Native gate and repository automation contracts |
| `delimited.go` | CSV delegate, chunk-safe record observer, source budget and cached header |
| `delimited_blank_record_contract_test.go` | Blank/comment record framing and chunk-boundary accounting |
| `delimited_test.go` | CSV syntax, header/errors, limits, arbitrary delimiters and timed ingestion |
| `doc.go` | Canonical ownership, supported adapters, lifecycle and cancellation |
| `edge_test.go` | Cross-reader invalid configuration and empty/boundary input |
| `encoding.go` | Explicit decode/UTF-8 validation and borrowed streaming source |
| `encoding_test.go` | Explicit encodings, streaming invalid UTF-8 and borrowed-source errors |
| `error_privacy_test.go` | Allowlisted diagnostics versus preserved private causes |
| `errors.go` | Stable classification, allowlisted diagnostics and retained private causes |
| `example_test.go` | Four rendered ingestion examples and explicit spreadsheet Close |
| `fixedwidth.go` | Pre-decode byte layout, Scanner limit and owned rows |
| `fixedwidth_test.go` | Byte layouts, decoding/shape errors, extraction fuzz and full-reader benchmark |
| `ingestion_error_contract_test.go` | Stable error categories, cause preservation and ingest coordinates |
| `internal/xls/admission.go` | Sparse admission versus workbook-wide dense allocation accounting |
| `internal/xls/admission_boundaries_test.go` | Workbook-wide dense slot boundaries and row gaps |
| `internal/xls/admission_contract_test.go` | Sheet/cell admission before materialization |
| `internal/xls/admission_test.go` | BIFF record admission and finite defaults |
| `internal/xls/biff.go` | BIFF subset, string continuation, numeric values and materialization |
| `internal/xls/biff_test.go` | Supported BIFF cell types, strings, numeric rendering and worksheet materialization |
| `internal/xls/difat_admission_test.go` | Physical DIFAT-chain count and sector admission |
| `internal/xls/mutation_boundaries_test.go` | CFB/BIFF malformed-input and arithmetic mutation regressions |
| `internal/xls/mutation_resource_guard_unix_test.go` | Conditional Unix mutation guard lifetime and ownership |
| `internal/xls/ole.go` | CFB geometry, wide counts, FAT/DIFAT chains and physical byte bounds |
| `internal/xls/ole_admission_width_test.go` | CFB count width and host-int conversion bounds |
| `internal/xls/ole_count_admission_test.go` | Declared sector counts versus physical source capacity |
| `internal/xls/ole_test.go` | FAT/mini-FAT, directory and workbook stream invariants |
| `large_benchmark_test.go` | Tagged generated workload, excluded creation and timed I/O/cleanup |
| `mutation_boundaries_internal_test.go` | Private parser-state and finite-accounting mutation boundaries |
| `mutation_resource_guard_unix_test.go` | Conditional Unix mutation guard lifetime and ownership |
| `normalize.go` | Ordered transformations, copied row/map boundaries and post-transform validation |
| `normalize_test.go` | Ordered trim/case/replacement and duplicate/empty-name policy |
| `reader_delegate_contract_test.go` | Delegate failures, shared cursor and cleanup contracts |
| `spreadsheet.go` | Common cursor, padding, error/presence policy, limits and Close |
| `spreadsheet_xls_test.go` | Real XLS fixture, selection/presence, finite materialization, fuzz and padded benchmark |
| `spreadsheet_xlsx.go` | Owned snapshot, preflight/delegate ordering and resource cleanup |
| `spreadsheet_xlsx_graph.go` | Complete XML document, canonical graph and delegate agreement |
| `spreadsheet_xlsx_presence.go` | Ordered row gaps, c-element presence and relationship identity |
| `spreadsheet_xlsx_presence_test.go` | Stored-empty versus absent cells and gap alignment |
| `spreadsheet_xlsx_test.go` | Raw XLSX values, error policy, lifecycle, OOXML fuzz and benchmark |
| `streaming_test.go` | Streaming consumption and reader lifetime |
| `xlsx_graph_admission_test.go` | Canonical XML graph, relationships and pre-delegate admission |
| `xlsx_presence_codec_test.go` | Registered ZIP codec boundaries and presence/source consistency |
| `xlsx_source_consistency_test.go` | Owned compressed snapshot versus mutable caller source |

## Disposition

All 47 tracked Go files (14 production, 33 assurance), 122 public members,
13 internal-exported members and 10 exported methods on private adapters are
audited. This includes types, named fields, functions, methods and constants;
private interfaces and their method contracts were inspected separately.

Go 1.27.2 native formatting passes all tracked sources. Native parser AST,
non-comment token streams (including inserted semicolons), build/generator and
security directives match the exact base. Both package go-doc outputs render
the clarified contracts. The native docs target passes all four examples and
its go vet operation; unchanged non-example runtime tests are not rerun.
Current source-doc checks report zero findings. API apidiff baseline bytes,
specification records and pinned fixtures remain unchanged. No runtime fix or new security/conformance guarantee is
introduced. No unresolved source-policy question has been identified. Broader
runtime, release/public-consumer, hosted CI and publication boundaries remain
separate from this Tier A pass.

## Objective Enforcement

The qualified public Tools v2.4.0 binary and immutable reusable CI source
provide the strict tracked-source check through `make source-docs-check`,
the existing local check/CI routes and Required aggregation. Release and
selected security/assurance routes retain their existing controls. This
structural status supplements the rendered semantic audit; it does not
certify prose meaning, generator execution or runtime conformance.

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
