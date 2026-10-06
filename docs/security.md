# Security

## Trust Boundary

Treat files, archive metadata, workbook relationships, encodings, delimiters,
headers, formulas, and row values as untrusted. Finite package defaults apply,
but callers should configure smaller limits where their schema permits and
retain upstream transport/file-size limits.

## Format Risks

ZIP and XLSX inputs can amplify compressed data. The ZIP source and expanded
data have finite defaults. XLS requires bounded
materialization. Delimited and fixed-width records can contain oversized or
invalid encodings. Formula values must not be treated as executable content.

## Application Responsibilities

The package does not provide upload authentication, malware scanning,
authorization, durable storage, or business-level schema validation. Reject
unexpected formats explicitly and avoid writing archive paths to disk.

See the [threat model](threat-model.md) and
[behavior and limits](behavior-and-limits.md). Report vulnerabilities through
[SECURITY.md](../SECURITY.md).
