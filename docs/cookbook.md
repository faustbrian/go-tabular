# Scenario cookbook

[Documentation index](README.md) · [Repository entry](../README.md)

## Finnish decimal CSV

Select `';'` as the delimiter. Values such as `12,50` remain strings; numeric
locale conversion belongs in application validation.

## Windows export

Wrap the source with `DecodeReader(source, EncodingWindows1252)` before
constructing a delimited reader. Fixed-width readers accept the encoding in
their own configuration.

## One record per line

For exports where every physical line is exactly one record, set
`SingleLineRecords` and `RejectBlankLines`:

```go
reader, err := tabular.NewDelimitedReader(source, tabular.DelimitedConfig{
	Delimiter:         ';',
	SingleLineRecords: true,
	RejectBlankLines:  true,
})
```

A quoted field spanning lines or an empty line, including a trailing one, is
then `ErrorMalformedRow` at that physical line.

## Fixed-width records with several certified widths

```go
reader, err := tabular.NewFixedWidthReader(source, tabular.FixedWidthConfig{
	Encoding:     tabular.EncodingISO88591,
	RecordWidths: []int{176, 220},
	Fields: []tabular.FixedWidthField{
		{Name: "postal_code", Start: 13, End: 18, TrimCutset: " \t\n\r\x00\v"},
	},
})
```

Any other record length is a malformed row, and the cutset also removes NUL
padding that `TrimSpace` would keep.

## Strict headers

Use trimming, a consistent case, exact replacements, `RejectEmpty`, and
`RejectDuplicates`. Keep the original header separately if audit output needs
to show the source spelling.

## Preserve spreadsheet errors

Set `PreserveCellErrors` only when strings such as `#DIV/0!` are legitimate
application data. The default rejects them with row and field coordinates.

## Inspect archive contents safely

Use `Entries`; it returns a copy in original central-directory order. Open an
exact name. Never concatenate entry names into destination filesystem paths.

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
