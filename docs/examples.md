# Executable examples

All supported examples are compiler-checked in
[`example_test.go`](../example_test.go) and run during the documentation gate.
They use the public package from `tabular_test`, so they exercise the same API
available to consumers.

Run every example:

```sh
go test github.com/faustbrian/go-tabular -run '^Example' -v
```

## Delimited and fixed-width input

- [`ExampleNewDelimitedReader`](../example_test.go) demonstrates a bounded
  semicolon-delimited import, header validation, `io.EOF`, and error handling.
- [`ExampleNewFixedWidthReader`](../example_test.go) demonstrates explicit byte
  ranges and whitespace trimming.

## ZIP-backed semicolon import

[`ExampleOpenZIP`](../example_test.go) opens a caller-owned archive source with
explicit entry, expanded-byte, compression-ratio, and symlink limits. It opens
and closes one exact entry and passes it to a bounded delimited reader.

## Spreadsheet import

[`ExampleOpenSpreadsheet`](../example_test.go) opens a caller-owned XLSX source,
selects a sheet, configures archive and parsed-value limits, validates the
header, reads a row, and closes the spreadsheet reader. Presence-sensitive
imports additionally set `PreserveCellPresence` and use `ReadCells` as
described in [behavior and limits](behavior-and-limits.md#spreadsheet-limits).
