# Behavior and limits

Zero-valued limits select safe defaults:

| Limit | Default |
| --- | ---: |
| Fixed-width record | 1 MiB |
| XLS/XLSX source workbook | 64 MiB |
| Delimited/spreadsheet record | 1 MiB |
| Delimited/spreadsheet field | 1 MiB |
| Delimited total source | 64 MiB |
| XLS/XLSX worksheets | 128 |
| XLS cumulative materialized slots | 1,000,000 |
| ZIP entries | 1,000 |
| ZIP compressed source | 256 MiB |
| ZIP single entry | 64 MiB |
| ZIP expanded total | 256 MiB |
| ZIP expanded/compressed ratio | 100:1 |

Applications should normally choose smaller values consistent with their
upload and job policies. The ZIP source limit is enforced before
central-directory parsing; expansion limits use central-directory declarations and
entry reads still verify CRC data. Unsafe absolute, parent, backslash, empty,
and duplicate entry names are rejected. Callers may increase or reduce the
ratio limit and reject symbolic links. Symbolic links are metadata-only by
default because this package never follows or writes ZIP entries to the
filesystem.

These payload limits are not heap guarantees. ZIP entry-count and XML sheet-count
checks follow directory/list materialization; XLS, XLSX validation and Excelize
may allocate substantially more than source bytes. Applications should combine package limits
with job-level memory and execution limits. See
[performance and memory verification](performance.md).

Delimited record and field limits use finite defaults. Applications reading
untrusted delimited sources should set smaller limits when their schema allows
them. Record limits
count the bytes presented to the parser, including delimiters and line endings.
Quoted multiline fields remain part of one logical record. The reader stops
the raw stream before the CSV parser can allocate past the configured record
bound. Field limits count parsed UTF-8 bytes before optional normalization.
Both failures use `ErrorLimitExceeded`; row and field coordinates are one-based
when available.

`MaxSourceBytes` counts all bytes consumed across Header and Read, including
blank lines and comments; zero selects 64 MiB. The exact cap followed by EOF is
accepted. One-over admission becomes a sticky typed limit error; complete rows
already delivered from the admitted prefix are not rolled back. Callers provide
interruptible I/O where cancellation is needed; the package cannot revoke a
blocking source Read.

XLS validates BIFF8 row/column ranges and charges `MaxMaterializedCells` across
every worksheet before materialization, including unselected sheets and at
least one slot per empty/absent row. Zero selects 1,000,000; positive values
override it. Sheet admission and repeated/overlapping offset rejection prevent
repeated decoding of one worksheet. This is an XLS-specific slot budget, not a
byte-accurate heap bound; XLSX uses streaming rows and does not use it.

Spreadsheet record and field limits also use finite defaults. They are enforced on
parsed XLS and XLSX cell values before normalization or caller delivery.
Record limits count the sum of bytes that would be delivered for one worksheet
row, including preserved spreadsheet error text. Field limits apply the same
rule to one cell and report its one-based coordinate.
XLSX callers may override the finite worksheet-count default before selecting
the first or named sheet. `MaxWorkbookBytes` applies to both XLS and XLSX source
sizes before either parser reads the workbook.
Archive and workbook limits still bound the underlying parser; parsed limits
do not claim to prevent allocations inside the XLS or Excelize engines.

`Read` intentionally exposes the compatibility `[]string` shape. Callers whose
business contract distinguishes a missing cell from an explicitly stored empty
cell must set `PreserveCellPresence` and use `ReadCells`. Each returned
`SpreadsheetCell` reports its normalized value and whether the workbook stored
that position. The option is disabled by default so string-only XLSX reads
retain their optimized cell-type lookup behavior and neither format allocates
per-cell presence storage. Enabled XLSX reads stream the selected worksheet XML
alongside Excelize so numeric and stored-empty trailing cells remain present.

Row normalization is ordered: trim whitespace, then replace empty values.
Header normalization removes a UTF-8 BOM from field one, trims, changes case,
applies exact replacements, then validates empty and duplicate names.

Fixed-width offsets refer to bytes in the original encoding. They must not
split a multi-byte UTF-8 character. ISO-8859-1 and Windows-1252 map every byte;
invalid UTF-8 is rejected instead of replaced.

With a header configured, the first record is consumed once and never returned
by `Read`. Without a header configuration, `Header` returns nil and consumes
nothing. Fixed field counts reject long rows and pad short spreadsheet rows;
variable mode preserves source width.
