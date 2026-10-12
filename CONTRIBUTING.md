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

Explicit full and release mutation gates must finish with zero surviving viable
mutants. Local mutation is optional, not a universal contribution prerequisite.

Do not fork shared verification workflows, add permanent replacements, machine-specific
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

## Selected Monthly Assurance

The owner-level [selected-assurance caller](.github/workflows/assurance.yml) is
configured for main only, on day 26 of each month at 06:13 UTC. It selects one
of the 6 current declared root-module fuzz targets using the shared UTC-week
rotation, with 30 seconds of active exploration. The selected risks are:
delimited/fixed-width parsing, byte extraction bounds, ZIP admission and
spreadsheet decoding under existing reader/resource policies.
The root targets use memory-backed input and checked-in spreadsheet fixtures;
they do not qualify every format/normalization or close/ownership boundary.
Existing interoperability fixtures, explicit full budgets and the manual
386 admission counterfactual retain their separate contracts.

Base-free ordinary checks execute deterministic seeds, not active exploration;
committed-range CI selects affected active smoke. Manual assurance requires an
exact anchored selector from the root fuzz operation in `.golib.yaml`, for
example `^FuzzDelimitedReader$`.
Empty, unknown and malformed selections refuse. The shared job bounds setup
and execution to ten minutes, active workers and CPU to two, owned memory to
2 GiB, native output to 1 MiB and private-cache lifetime. Failure, cancellation,
exhausted budgets and cleanup refusal remain attributable failures. This lane
does not run mutation or replace Required, original full fuzz budgets,
explicit `make check-full`, or relevant release qualification.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.

## Source Documentation

Install the public `golib` v2.4.0 verifier declared in `.golib.yaml` and
validate its declared checksum before repository checks. Run
`make source-docs-check` for the tracked-source documentation contract.
`make check`, `make check-full` and `make ci` require the same strict check.
Required CI also checks all tracked source, including inactive build variants.
Manual-review findings stay visible; structural success does not certify prose
truth, specification conformance or generator reproducibility.
