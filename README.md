# go-tabular

`go-tabular` is a production-oriented Go package for explicit, bounded
tabular ingestion. It handles CSV and other delimiters, fixed-width text,
legacy XLS, XLSX, and ZIP-backed source files without format auto-detection or
implicit data conversion.

```sh
go get github.com/faustbrian/go-tabular
```

```go
reader, err := tabular.NewDelimitedReader(source, tabular.DelimitedConfig{
    Delimiter: ';',
    Header: &tabular.HeaderConfig{
        TrimSpace:        true,
        Case:             tabular.HeaderCaseLower,
        RejectEmpty:      true,
        RejectDuplicates: true,
    },
})
if err != nil {
    return err
}

header, err := reader.Header()
if err != nil {
    return err
}
for {
    row, err := reader.Read()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        return err
    }
    consume(header, row)
}
```

## Design guarantees

- Formats and encodings are selected explicitly; there is no auto-detection.
- Delimited, fixed-width, ZIP entry, and XLSX row processing are streaming.
- XLS is bounded but materialized because OLE2/BIFF8 requires random access.
- ZIP archives are indexed only after entry-count, size, path, and duplicate
  checks.
- UTF-8 is validated; supported legacy encodings are converted explicitly.
- Normalization is opt-in and returns new rows instead of mutating inputs.
- Stable error kinds work with `errors.Is`; row and field coordinates are
  one-based.

## Format summary

| Format | Processing | Supported core | Important boundary |
| --- | --- | --- | --- |
| CSV/delimited | Streaming | quotes, comments, delimiters, headers | caller supplies encoding conversion |
| Fixed-width | Streaming | byte ranges, trimming, three encodings | offsets are bytes before decoding |
| XLS | Bounded materialization | OLE2 + common BIFF8 cell records | no macros, formulas, formatting, or editing |
| XLSX | Streaming rows | raw values, errors, sheet selection | OOXML ZIP is validated before Excelize |
| ZIP | Streaming entries | exact lookup and extraction | no recursive extraction or filesystem writes |

See the [full format matrix](docs/formats.md) and
[behavior and limits](docs/behavior-and-limits.md) before adopting the
spreadsheet readers.

## Documentation

- [Quickstart](docs/quickstart.md)
- [Architecture](docs/architecture.md)
- [Public API reference](docs/api.md)
- [Performance and memory verification](docs/performance.md)
- [Adoption guide](docs/adoption.md)
- [End-to-end examples](docs/examples.md)
- [Scenario cookbook](docs/cookbook.md)
- [FAQ](docs/faq.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Migration notes](docs/migration.md)
- [Versioning and releases](docs/versioning.md)
- [Contributing](CONTRIBUTING.md) and [security policy](SECURITY.md)

## Dependency and provenance policy

XLSX support uses
[`github.com/xuri/excelize/v2`](https://github.com/qax-os/excelize). The XLS
reader is maintained in `internal/xls`: it is a deliberately reduced,
hardened OLE2/BIFF8 implementation informed by `github.com/millken/xls`, not a
module dependency. Its Apache-2.0 provenance is retained in
[`internal/xls/NOTICE.md`](internal/xls/NOTICE.md). The repository is
distributed under the [Apache License 2.0](LICENSE).

## Status

The API is pre-v1. Supported behavior is fixture-backed, fuzzed, benchmarked,
and held to 100% production-statement coverage. Export helpers are not part of
the first release; see the [roadmap](ROADMAP.md).
