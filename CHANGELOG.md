# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and releases follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

- Replace copied repository-local verification tooling with the released
  `go-library-tools` v1.2.0 specification-governance workflow while preserving
  package-owned fixtures, mutation evidence, API compatibility, fuzzing,
  benchmark, and documentation gates.

### Documentation

- Clarify how shared safety-policy updates are coordinated across standalone
  repositories.

- Replace archived monorepo links and completed execution artifacts with a
  standalone, human-oriented documentation structure.

## [1.0.0] - 2026-08-25

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

[Unreleased]: https://github.com/faustbrian/go-tabular/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/faustbrian/go-tabular/releases/tag/v1.0.0
