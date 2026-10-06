# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and releases follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.1] - 2026-10-06

### Maintenance

- Update the spreadsheet parser's x/text dependency to v0.42.0 for Unicode
  normalization correctness fixes while retaining the root's Go 1.27 floor.
- Refresh immutable CI workflow and runner-action pins while preserving
  checksum-selected tooling and the separate root release qualifier.
- Align installation and adoption guidance with the already published v2
  module, without claiming downstream adoption.

## [2.0.0] - 2026-10-06

The root source tree uses the `/v2` module path. Public tags and releases
establish availability; release gates precede publication, and maintained
consumer adoption follows verification of the public module. Historical v1
consumers retain their published behavior without local `replace` directives.

### Maintenance

- Adopt the checksum-pinned public Golib v1.8.5 CLI and an explicit source-built
  root release qualification route. Four independently reviewed XLS boundary
  equivalents retain their native status under exact source/verifier binding;
  all other viable mutation findings still fail qualification. These controls
  prepare root v2 qualification without asserting publication or a passing
  release verdict.

### Security

- Preserve XLS physical sector-count capacity without narrowing host-sized
  lengths, and refuse out-of-range wire sector IDs before host-width indexing.
  Propagate missing physical stream-sector errors instead of substituting
  zero-filled bytes, without publishing a partially admitted workbook.
  Valid workbook and RK value semantics remain unchanged on 32- and 64-bit hosts.
- Reject XLSX sheet names, application relationship IDs and targets rewritten
  by the pinned parser's namespace translation, preserving admitted worksheet
  identity.
  Preserve strict namespace declarations and relationship-type vocabulary.
- Require one namespace-qualified worksheet relationship identity, refusing
  unrelated or ambiguous identity attributes before delegate parsing.
- Snapshot XLSX sources within workbook and archive byte limits before
  admission, parsing, and presence-aware iteration, preventing mixed source
  revisions. Reject incomplete declared sources and retain private I/O causes.
- TABULAR-DEC-002 sha256:193d424eb2d2d034843cebb1243585b0dd8acfa9b5c37ef8e978653a09d6d917
- TABULAR-DEC-005 sha256:a93f6c978d4caae3bd80d5c4b8e6f23b5d256840e98e7f17be5e6074c7f01776
- TABULAR-DEC-008 sha256:fc27a152f2251c2b4546c1ba0dbc2ea1d16f3e9668cc5f64d97cd337eb52a619
- Omit underlying parser, source, destination, and requested-entry diagnostics
  from default `Error` messages. Messages retain fixed category/context and
  coordinates; explicit `Err`, `errors.Is`/`As`, and `Unwrap` inspection preserve
  the original cause and require trusted handling.
- Apply finite defaults to delimited and spreadsheet row and field parsing,
  XLSX source size and worksheet counts, and ZIP source, expansion size, and
  ratio to constrain accepted input. Explicit positive limits continue to override
  defaults; input limits are not exact heap or I/O deadline guarantees.
- Add `DelimitedConfig.MaxSourceBytes` (64 MiB default) across header and row
  reads, including skipped comments and blank lines. Preserve comment/quoted
  multiline grammar with split reads and Unicode comment markers.
- Validate BIFF8 dimensions and admit XLS sheets plus cumulative materialized
  slots before dense/presence allocation. `MaxMaterializedCells` defaults to
  1,000,000 across all sheets and charges empty/absent row slots; `MaxSheets`
  now applies to XLS as well as XLSX. Repeated/overlapping sheet offsets fail.
- Reject external and unsafe XLSX worksheet relationships in both string-only
  and presence-aware modes.
- Admit OLE FAT/DIFAT counts against physical sectors before allocating indexes.
  Reject ambiguous or noncanonical XLSX workbook graphs and validate complete
  selected worksheet documents, including safely relocated worksheet parts.
- This security change is prepared for v2 because it changes documented
  zero-value behavior and adds source/materialization budget fields. Migrate
  unkeyed `ZIPConfig`, `DelimitedConfig`, and `SpreadsheetConfig` literals to
  keyed fields and set explicit positive limits
  when accepted payloads exceed the new defaults.
- TABULAR-DEC-002 sha256:bec8c6ce4f29d17394f31b0c850c1ad16e81ac80d3e831234d86fd9c10d7af33
- TABULAR-DEC-008 sha256:de1633d5ee9a84bb8c1860a3a8226e869f99ffb0c14cccba710e66d09391513b

## [1.1.0] - 2026-10-04

### Compatibility

- Require Go 1.27.0 or later, up from Go 1.26.6. Upgrade local and CI
  toolchains before adopting this version. The module path and public API
  remain unchanged.

### Specification Decisions

- Publish the [specification decision register](docs/specification-decisions.md),
  pinned authorities, conformance bindings, monitoring, and append-only history
  for CSV, XLS, and XML/ZIP-backed XLSX ingestion boundaries.
- TABULAR-DEC-001 sha256:5fd9461c00ba1d4c76d539666f3281ffb1062bcc49cf3cbf491143b5292870ad
- TABULAR-DEC-002 sha256:f9ee61e1ce837b4d533539705dfb59ae3b5dd03ee5489e3b5ce947ef552a4132
- TABULAR-DEC-003 sha256:c4c188ea089bd489be9204a4c6e642761cb28ffc5b859f94aa1e54f487768652
- TABULAR-DEC-004 sha256:57ad86da0ea0669cbe31986f82d7b1ee86b568a3161e0a4aa7db5995f803821b
- TABULAR-DEC-005 sha256:5d94938190a0017bdd9b95c45b6b0856b079952e934d80e002c7f61781cc2521
- TABULAR-DEC-006 sha256:ee666df9d8ec0ff1c0a889d19237510505ee986b16ef18ffb2d071c088e8601e
- TABULAR-DEC-007 sha256:e7b03868875fab2883c7479e1622d8871f259e79d53889626805e0acdcdbfa21
- TABULAR-DEC-008 sha256:8b8e401a301548e4ccd3c47454f773dbbcacadcf3c97f1ae2517c71f1389699e

### Maintenance

- Update the text-processing dependency to `golang.org/x/text` v0.41.0.

- Upgrade the checksum-pinned `go-library-tools` CLI and reusable CI workflow
  to immutable v1.4.0 W14 enforcement while retaining the schema-v2 cohesion
  contract, local `make cohesion` entry point, package-owned gates, and online
  specification authority monitoring in `make ci`.

- Adopt the `go-library-tools` v1.3.0 schema-v2 cohesion contract and local
  `make cohesion` gate without changing the tabular API or runtime behavior.
- Pin reusable CI to the v1.3.0 workflow and enforce cohesion metadata in the
  repository's required CI contract.

- Replace copied repository-local verification tooling with the released
  `go-library-tools` v1.2.0 specification-governance workflow while preserving
  package-owned fixtures, mutation evidence, API compatibility, fuzzing,
  benchmark, and documentation gates.

### Documentation

- Complete the package documentation contract with compiler-checked quick
  starts for delimited, fixed-width, ZIP-backed, and spreadsheet ingestion;
  explicit package, lifecycle, cancellation, resource, and concurrency
  ownership; accurate release guidance and delivery metadata; direct support
  and security routes; and the correct Apache-2.0 badge.
- Replace the completed first-release roadmap with the published v1 boundary,
  and correct the `v1.0.0` release date to its 2026-08-26 publication date.

- Record the behavior-neutral 2026-09-06 re-review of unchanged MS-CFB 12.0,
  MS-XLS 12.2, and MS-OI29500 25.0 normative content after further Microsoft
  landing-page presentation changes.

- Correct the repository standards to Go 1.26.6 and replace the pre-release
  export roadmap wording with the published v1.0.0 boundary.

- Record the behavior-neutral 2026-09-04 re-review of unchanged Microsoft
  format PDFs after recurring landing-page presentation changes, retaining the
  existing TABULAR-DEC-004 through TABULAR-DEC-007 decisions and bindings.

- Link ecosystem and Integration and data movement family guidance to the
  immutable v1.4.0 documentation release.

- Record the behavior-neutral review of unchanged Microsoft format PDFs and
  PKWARE APPNOTE 6.3.10 content. Refresh the Microsoft landing-page monitoring
  digests and replace the mutable PKWARE product page with its current general
  APPNOTE payload as the release-change monitor for TABULAR-DEC-004 through
  TABULAR-DEC-008.

- Publish the module's family, capabilities, ownership, lifecycle, supported
  environments, package selection, and delivery status, and link the README to
  the immutable v1.3.0 ecosystem index and family guidance.

- Clarify how shared safety-policy updates are coordinated across standalone
  repositories.

- Replace archived monorepo links and completed execution artifacts with a
  standalone, human-oriented documentation structure.

## [1.0.0] - 2026-08-26

### Changed

- Validate action pinning from the standalone repository root and leave
  repository-foundation policy to the authoritative repository contract.

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Documentation

- Replace obsolete standalone-repository links and workflow claims with
  monorepo-canonical targets and current release guidance.

- Link the package README to package-owned documentation.

### Compatibility

- Added a pinned module export baseline so incompatible public API changes
  fail the canonical repository gate.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-tabular` identity while preserving its documented API and behavior.
- Added the `GO-SAFETY-1` ownership, concurrency, race, fuzz, resource, and
  benchmark standard with an executable `make safety` gate.
- Moved AI planning and hardening briefs into `.ai/` and clarified the
  separate purposes of ownership notices and detailed source provenance.

### Added

- Added opt-in pre-allocation logical-record limits and parsed-field limits to
  `DelimitedConfig`, including quoted multiline records and stable
  `ErrorLimitExceeded` classification. Zero values preserve the existing
  unbounded behavior.
- Added opt-in parsed-row and cell limits to `SpreadsheetConfig` for XLS and
  XLSX. Limit failures report `ErrorLimitExceeded`; zero values preserve
  existing behavior.
- Added opt-in ZIP compression-ratio and symbolic-link policies plus an XLSX
  worksheet-count limit. Zero values preserve existing archive and workbook
  behavior.
- Added opt-in spreadsheet cell-presence preservation through
  `PreserveCellPresence` and `ReadCells`, distinguishing absent positions from
  explicitly stored empty cells while keeping presence storage and ordinary
  XLSX cell-type lookups off the default `Read` path.
- A standardized OSS repository skeleton covering policy, documentation,
  legal notices, Go tooling, pinned CI, security, and release automation.
- Gated, disk-backed benchmarks for CSV and XLSX inputs of at least 50 MiB and
  100,000 rows, including a scheduled workflow with peak-memory reporting.
- Explicit chunked-streaming regressions, malformed fixtures for every major
  format, and documentation of benchmark inputs and XLSX heap amplification.
- Production readers for CSV/delimited, fixed-width, XLS, XLSX, and ZIP-backed
  ingestion with explicit limits, normalization, and structured errors.
- Realistic fixtures, hostile-input regressions, format fuzz targets,
  representative benchmarks, and 100% production-statement coverage.
- Adoption, API, architecture, format, behavior, troubleshooting, migration,
  versioning, and scenario documentation.
- Initial package goals for `tabular`, covering CSV, XLS, XLSX,
  fixed-width, and ZIP-backed ingest as the first supported scope.
- Hardening goals covering hostile-input handling, encoding discipline,
  fixture quality, performance validation, and meaningful 100% coverage.
- Package maintenance rules enforcing changelog hygiene, SemVer treatment of
  public APIs, and meaningful 100% coverage for production code.

### Fixed

- Keep module-archive tests scoped to files shipped with the tabular module;
  repository-root workflow policy remains owned by the root verification gate.
- Avoid redundant row copies when CSV normalization is disabled and use a
  bounded 64 KiB source buffer to improve large-file throughput.
- Bound fuzz-smoke concurrency to avoid deadline flakes on high-core hosts.
- Avoid per-cell XLSX type lookups for ordinary values, substantially reducing
  runtime, allocations, and peak memory for large workbooks.
- Classify corrupt ZIP entry read failures through `ErrorArchive` while
  preserving the standard library's declared-size boundary.

[Unreleased]: https://github.com/faustbrian/go-tabular/compare/v2.0.1...HEAD
[2.0.1]: https://github.com/faustbrian/go-tabular/compare/v2.0.0...v2.0.1
[2.0.0]: https://github.com/faustbrian/go-tabular/releases/tag/v2.0.0
[1.0.0]: https://github.com/faustbrian/go-tabular/releases/tag/v1.0.0
