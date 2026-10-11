# Contributing

## Before Editing

1. Read [`AGENTS.md`](AGENTS.md) and the affected module's docs.
2. Run standalone `go test ./...` or a focused affected-package test.
3. Identify owned dependencies and reverse dependants in `modules.json`.
4. Preserve unrelated work and generated/corpus provenance.

## Changes

Keep commits focused and conventional. Update every affected changelog with
the behavior and migration impact. Public API changes require compatibility
evidence and documentation. Specification behavior requires a decision record,
fixture coverage, and interoperability evidence.

New direct dependencies and dependency updates must follow the
[dependency governance policy](AGENTS.md#dependencies-and-supply-chain). Package-local
update bots are forbidden; the root policy owns every module and action update.

Specification-backed changes must follow the
[specification governance contract](AGENTS.md#design), update
the affected entries in the [specification decision register](docs/specification-decisions.md),
and complete the Specification Decisions section of the pull request template.
An unresolved interpretation or stale source pin is release-blocking; peer
behavior cannot silently select policy.

Full scheduled and release mutation gates must finish with zero surviving viable
mutants. Local mutation is optional, not a universal contribution prerequisite.

Do not add package-local workflows, permanent replacements, machine-specific
paths, bypass flags, broad mutation exclusions, or aggregate quality metrics
that hide a failing package.

## Verification

Standalone `go test ./...` runs deterministic tests, fuzz seeds and saved corpus
regressions without sibling checkouts or an ecosystem goal ledger. Readers are
caller-serialized; focus parser verification on resource-limit, close/ownership,
format and normalization assertions and affected compatibility fixtures.

With the checksum-pinned public verifier selected by `.golib.yaml`, use:

```bash
make inventory
make check
make ci
```

`check` and `ci` select bounded local checks; the manifest requires no service or
test tag. Ordinary CI consumes the public binary, not a sibling Tools checkout.
Short active exploration selects one target; optional local mutation selects
one package:

```bash
GOMAXPROCS=2 go test -run '^$' -fuzz '^FuzzDelimitedReader$' -fuzztime=2s -parallel=1 -timeout=2m .
golib mutation --module . --package .
```

`make check-full` retains full assurance, including online specification checks,
all six 10000-execution fuzz campaigns, interoperability fixtures and benchmarks.
`make specification-check-online` is also available explicitly. Release
qualification retains the [release preconditions](docs/releasing.md), public
consumer boundary and selected manual 386 admission counterfactual. A local pass
is not full assurance, hosted-CI or release qualification; report unavailable,
skipped and failed boundaries honestly.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.
