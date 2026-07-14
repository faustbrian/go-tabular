# Contributing

Thank you for helping make `go-tabular` a dependable tabular-ingest
foundation.

## Before opening a change

Use an issue for behavior changes that affect public APIs, parsing semantics,
normalization rules, supported formats, compatibility, or performance
characteristics. State which category the proposal belongs to:

- supported-format behavior;
- ingest or normalization semantics;
- compatibility or API design;
- performance or memory behavior;
- documentation or examples;
- implementation or test hardening.

Do not present service-specific ingestion conventions as generic package
requirements without showing why they belong in the package.

## Development setup

Requirements:

- Go 1.25 or later;
- Git;
- `golangci-lint` for the same lint gate used by CI.

Clone the repository and run:

```sh
go mod download
go test ./...
go vet ./...
```

## Change requirements

- Add a regression test before fixing a defect.
- Keep meaningful 100% production-code coverage.
- Preserve deterministic parsing and normalization behavior unless a
  documented breaking change is intentional.
- Update examples and user documentation for public behavior changes.
- Update `GOAL.md` / `GOAL_HARDEN.md` when the package scope or hardening bar
  changes materially.
- Add an entry under `Unreleased` in `CHANGELOG.md`.
- Keep dependencies minimal and explain every addition.

## Local verification

```sh
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
go test -coverpkg=./... -coverprofile=/tmp/go-tabular-coverage.out ./...
go tool cover -func=/tmp/go-tabular-coverage.out
go test ./... -run '^Example'
go test ./... -run '^$' -bench . -benchtime=100ms
```

Run fuzz targets for parsing changes where relevant.

## Commit and pull request style

Use focused conventional commits with a body explaining why the change is
needed. Pull requests should include:

- the ingest or parsing problem being solved;
- compatibility and normalization impact;
- tests and fixtures added;
- verification commands and results;
- documentation and changelog updates.

## Reporting security issues

Do not open a public issue for a suspected vulnerability. Follow
[SECURITY.md](SECURITY.md).
