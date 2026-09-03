# Tabular specification governance

The package combines externally defined file formats, maintained parser
delegates, and repository-owned projection and resource policy. The human
decision register is [`../docs/specification-decisions.md`](../docs/specification-decisions.md);
the machine bindings are [`decisions.json`](decisions.json),
[`conformance.json`](conformance.json), [`decision-history.json`](decision-history.json),
and [`monitoring.json`](monitoring.json). Source and fixture provenance is pinned
in [`manifest.json`](manifest.json).

| Decision | Conformance boundary |
| --- | --- |
| TABULAR-DEC-001 | Delegated CSV grammar |
| TABULAR-DEC-002 | Explicit delimited profiles and limits |
| TABULAR-DEC-003 | Header, shape, and normalization policy |
| TABULAR-DEC-004 | OLE compound workbook boundary |
| TABULAR-DEC-005 | Strict BIFF8 ingestion subset |
| TABULAR-DEC-006 | Delegated XLSX raw-value decoding |
| TABULAR-DEC-007 | XLSX relationships and cell presence |
| TABULAR-DEC-008 | Spreadsheet and ZIP resource policy |

The millken XLS workbook is provider-origin fixture evidence. Go
`encoding/csv` and Excelize are production delegates. None of those is recorded
as an independently executed maintained-peer differential.

The Microsoft publication PDFs exceed the online checker's bounded authority
size, so `manifest.json` pins their exact bytes while `monitoring.json` checks
the bounded Microsoft publication and release pages for upstream change.

Append-only upstream dispositions are recorded in
[`review-history.md`](review-history.md).
