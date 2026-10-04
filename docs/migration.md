# Migration notes

Public API adjustments follow Semantic Versioning and are recorded in the changelog.
Pin a tagged version rather than `main`.

## Upgrading to v1.1.0

Version `v1.1.0` requires Go 1.27.0 or later; `v1.0.0` requires Go 1.26.6.
Upgrade local and CI toolchains before updating the module. Keep the existing
`github.com/faustbrian/go-tabular` import path; the public API is unchanged.

## Migrating from other libraries

When migrating from `encoding/csv`, note that `NewDelimitedReader` requires an
explicit delimiter, header processing is opt-in, normalization returns copies,
and parser errors are wrapped in stable tabular kinds.

When migrating from direct Excelize use, this package intentionally exposes
only ordered strings, cell-error policy, sheet selection, limits, and common
row semantics. Code requiring styles, formulas, merged cells, or editing
should continue to use a spreadsheet-specific API.

When replacing another XLS library, validate real BIFF8 fixtures. The internal
reader supports a documented subset and rejects unsupported or corrupt
structures rather than attempting recovery.
