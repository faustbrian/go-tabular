# Supported-format matrix

[Documentation index](README.md) · [Repository entry](../README.md)

| Capability | CSV/delimited | Fixed-width | XLS | XLSX | ZIP |
| --- | --- | --- | --- | --- | --- |
| Streaming records | Yes | Yes | No | Yes | N/A |
| Explicit headers | Yes | Field names | Yes | Yes | N/A |
| Variable rows | Configurable | Short records only | Configurable | Configurable | N/A |
| Cell error handling | N/A | N/A | Reject/preserve | Reject/preserve | N/A |
| Sheet selection | N/A | N/A | Exact name | Exact name | N/A |
| Encoding conversion | Caller | Built in | BIFF8 | OOXML UTF-8 | Opaque bytes |
| Resource limit | Surrounding reader | Record bytes | Workbook bytes | ZIP limits | ZIP limits |

## XLS support

The internal reader supports OLE2 compound files containing BIFF8 workbook
streams, shared strings (including continuations), rows, labels, numbers, RK
numbers, multiple RK values, booleans, and error cells. It does not evaluate
formulas, preserve formatting, execute macros, edit workbooks, or promise
coverage for pre-BIFF8 files.

## XLSX support

XLSX returns raw cell values through Excelize. Worksheet XML must be
well-formed and the OOXML ZIP must satisfy configured limits. Formula
evaluation, formatting, charts, macros, and workbook editing are outside the
package contract.

## Delimited support

Quoting, escaped quotes, embedded delimiters, and quoted newlines follow Go's
`encoding/csv`. Delimiters are single runes. Auto-detection and multi-character
delimiters are intentionally unsupported.

Line-oriented exports can opt into one physical line per record.
`SingleLineRecords` rejects a quoted field that spans lines, and
`RejectBlankLines` rejects empty lines that `encoding/csv` would skip, before,
between or after records. Both report a malformed row at the physical line; a run of blank lines is
reported once, at its first line. An unterminated quote still consumes the
rest of the input within the record and source limits.
Blank-line rejection cannot be combined with comments.

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
