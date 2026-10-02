# tabular

[![CI](https://github.com/faustbrian/go-tabular/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/faustbrian/go-tabular/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/badge/CodeQL-required-blue)](https://github.com/faustbrian/go-tabular/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Mutation](https://img.shields.io/badge/mutation-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Documentation](https://img.shields.io/badge/docs-checked_in_CI-blue)](docs/)
[![Go Reference](https://pkg.go.dev/badge/github.com/faustbrian/go-tabular.svg)](https://pkg.go.dev/github.com/faustbrian/go-tabular)
[![Release](https://img.shields.io/github/v/release/faustbrian/go-tabular?sort=semver)](https://github.com/faustbrian/go-tabular/releases)
[![Go](https://img.shields.io/badge/go-1.27.0-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

`tabular` provides explicit, bounded ingestion for CSV and other
delimiters, fixed-width text, legacy XLS, XLSX, and ZIP-backed sources without
format auto-detection or implicit data conversion.

## Status

This source tree defines `github.com/faustbrian/go-tabular/v2` with finite
security defaults. Public tags and releases establish version availability;
historical v1 retains its original defaults. Release gates and subsequent
public-module consumer adoption are separate boundaries. Production code is
held to meaningful 100% statement coverage.

## Requirements

- Go 1.27.0 or later
- portable Go targets supported by the Go toolchain; no cgo or external service
  is required

## Installation

```sh
go get github.com/faustbrian/go-tabular@v1
```

The hardened v2 source is not available from the public module proxy until a
`v2` tag is published. Existing applications must remain on v1 and configure
explicit positive limits until then.

## Quickstart

The compiler-checked
[`ExampleNewDelimitedReader`](example_test.go) creates a bounded semicolon
reader, validates its header, reads every row through `io.EOF`, and demonstrates
stable error handling. Run it directly after installation:

```sh
go test github.com/faustbrian/go-tabular -run '^ExampleNewDelimitedReader$' -v
```

The [five-minute quickstart](docs/quickstart.md) explains the same complete
flow. Compiler-checked [examples](docs/examples.md) also cover fixed-width,
ZIP-backed, and spreadsheet ingestion.

## Package Guarantees

- explicit format and encoding selection
- streaming delimited, fixed-width, ZIP-entry, and XLSX row processing
- bounded XLS materialization for OLE2/BIFF8 random access
- archive source-size, entry-count, expansion, compression-ratio, path, link,
  and duplicate checks
- finite defaults for XLSX worksheet counts and parsed delimited/spreadsheet rows
- opt-in absent-versus-stored-empty spreadsheet cell preservation
- opt-in normalization that does not mutate caller-owned rows
- stable error kinds with one-based row and field coordinates

See [formats](docs/formats.md) and
[behavior and limits](docs/behavior-and-limits.md) for exact boundaries.

## Package Map And Ownership

- The root module is the only public package. It owns explicit tabular
  decoding, validation, limits, normalization, and stable error categories.
- `internal/xls` is an implementation detail for the documented BIFF8 subset;
  consumers cannot import it and it is not a compatibility surface.

The package never closes caller-provided sources. Delimited and fixed-width
readers retain their `io.Reader`, and `ZIPArchive` retains its `io.ReaderAt`.
XLS input is fully consumed by `OpenSpreadsheet`; XLSX captures a bounded owned
revision during construction, shared by admission, parsing, and presence-aware
iteration. Sources may be released after successful construction. Callers must
close spreadsheet
readers and readers returned by `ZIPArchive.Open`. The package starts no
goroutines, performs no retries, and has no hidden shutdown phase. Reader
instances are stateful and are not safe for concurrent method calls; callers
own serialization.

Constructors take explicit configuration values, validate them before parsing,
and copy retained mutable fields. Zero values select documented finite archive,
workbook, worksheet, record, and field limits. Applications should set smaller
positive limits when their ingest policy is narrower than those defaults.

Parsing is deliberately context-free: operations synchronously consume the
caller-provided `io.Reader` or `io.ReaderAt` and do not perform network I/O.
Cancellation and deadlines therefore belong to that source and the calling
goroutine. This is the ecosystem's frozen context-free codec exception, not a
promise that arbitrary source reads can be interrupted by this package.

## Documentation

Start with the [documentation index](docs/README.md), [quickstart](docs/quickstart.md),
[adoption guide](docs/adoption.md), and [API reference](docs/api.md). Review
[performance](docs/performance.md), [security](docs/security.md), and
[behavior and limits](docs/behavior-and-limits.md) before accepting hostile files.

Use `tabular` for explicit, bounded import from a known format. Do not use it
for format detection, schema inference, value conversion, export, persistence,
or workflow orchestration. There are no public subpackages, adapters, companion
modules, or testing helpers in this repository.

For ecosystem-wide selection and ownership guidance, see the versioned
[Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/README.md)
and its [Integration and data movement family](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/design-language.md#package-families-and-selection).

Release history is maintained in [CHANGELOG.md](CHANGELOG.md).
Specification-backed behavior and delegated parser boundaries are recorded in
the [specification decision register](docs/specification-decisions.md).
Operational and compatibility details are in [troubleshooting](docs/troubleshooting.md),
[FAQ](docs/faq.md), [migration](docs/migration.md), and
[compatibility](COMPATIBILITY.md).

## Development

Run `make check` before submitting a change. This enforces formatting, static
analysis, race tests, meaningful 100% coverage, parser fuzz smoke, benchmarks,
documentation, and vulnerability scanning.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) and follow the
[code of conduct](CODE_OF_CONDUCT.md). Format and normalization changes require
explicit compatibility and data-integrity analysis.

## Security

Report vulnerabilities privately according to [SECURITY.md](SECURITY.md).
Review [docs/security.md](docs/security.md) before ingesting untrusted files.
Use [GitHub Issues](https://github.com/faustbrian/go-tabular/issues) for defects
and adoption questions; see [SUPPORT.md](SUPPORT.md) for reporting guidance.

## License

`tabular` is available under the [Apache License 2.0](LICENSE). XLS
provenance and third-party attribution are recorded in [NOTICE](NOTICE) and
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
