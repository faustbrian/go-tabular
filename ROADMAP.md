# Roadmap

## Published v1 foundation

`v1.0.0` was published on 2026-08-26 with the CSV, delimited, fixed-width,
XLS, XLSX, and ZIP-backed ingest surface. Its release established the current
compatibility promise, security boundary, and benchmark baseline.

The repository currently has CI but no tag-triggered release workflow or local
release command. Maintainers publish releases manually under
[`docs/releasing.md`](docs/releasing.md); adding automation remains separate
future work rather than delivered functionality.

## After core stabilization

Potential work is evaluated independently and is not promised for `v1`:

- TSV-first convenience helpers;
- export helpers;
- schema mapping helpers;
- archive-format expansion beyond ZIP;
- carefully justified additional tabular formats.

The core will not become a workflow engine, ETL platform, queue layer, or
application-specific transformation framework.
