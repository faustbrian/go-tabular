# Repository Standards

This repository follows the shared maintenance baseline used by the
`faustbrian/go-*` OSS packages.

## Mandatory Root Files

Every repository contains `.gitattributes`, `.gitignore`,
`.golangci.yml`, `AGENTS.md`, `CHANGELOG.md`, `CLAUDE.md`,
`CODE_OF_CONDUCT.md`, `CONTRIBUTING.md`, `LICENSE`, `Makefile`, `NOTICE`,
`README.md`, `ROADMAP.md`, `SECURITY.md`, and `THIRD_PARTY_NOTICES.md`.

Completed implementation plans and verification snapshots belong in issue
tracking or Git history rather than the released source tree.

`NOTICE` identifies project and inherited ownership. `THIRD_PARTY_NOTICES.md`
separately records detailed source provenance and third-party attribution.
Both remain present even when no additional third-party source requires
attribution. A package may retain a different approved OSS license when
provenance requires it.

## Mandatory Documentation

The shared taxonomy is lowercase kebab-case and includes a documentation
index, quickstart, usage guidance, API reference, architecture, examples,
cookbook, FAQ, troubleshooting, migration, compatibility, performance,
security, Go safety and concurrency, and releasing guide.
Package-specific documents extend this taxonomy without renaming shared
concepts.

## Mandatory Automation

This repository uses one pinned reusable CI workflow for pull requests, main,
scheduled verification, and manually dispatched release rehearsal. It does not
currently automate tag or release publication. CI tests Go 1.27.0 as the
supported minimum. The shared gate includes dependency and reachable
vulnerability checks.

The public Make interface is `check`, `ci`, `cohesion`, `docs`, `inventory`,
`repository-check`, `specification-check`, and `workflows`. The shared `golib`
tool owns the underlying format, test, race, coverage, lint, fuzz, mutation,
documentation, API, security, and benchmark gates.

The package family shares the `GO-SAFETY-1` baseline. It forbids `unsafe`,
cgo, and `go:linkname` in production code and standardizes ownership,
goroutine lifecycle, race, fuzz, resource-bound, leak, and benchmark evidence.

## Approved Package-Specific Differences

- `jsonapi` carries JSON:API feature, conformance, extension/profile,
  recommendation, and threat-model documentation.
- `jsonrpc` carries protocol conformance and middleware documentation.
- `queue` carries backend, delivery, lifecycle, failure, and integration
  documentation plus a live-backend integration workflow. Its fork provenance
  requires detailed third-party notices.
- `wire` carries format, dependency, and audit-evidence documentation.
- `tabular` carries format and ingest-limit documentation. It uses
  Apache-2.0 and retains XLS provenance notices.

Code, dependencies, fuzz targets, benchmark inputs, and domain-specific
security guidance are expected to differ. Shared policy wording and automation
structure must not drift without updating this contract across all affected
repositories.
