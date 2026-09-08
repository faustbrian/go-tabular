# Releasing

## Preconditions

Release from a clean, synchronized `main` branch. Add one dated release
section to `CHANGELOG.md`, document compatibility or migration impact, and
review `NOTICE` plus `THIRD_PARTY_NOTICES.md` whenever XLS provenance or
dependencies change.

The repository does not currently provide a release command or a tag-triggered
release workflow. Release publication is a maintainer-owned manual operation;
do not infer automation from the reusable CI workflow.

## Verification

```sh
make check
```

The release is blocked unless format checks, race tests, meaningful 100%
coverage, parser fuzz smoke, benchmarks, documentation links, lint, and
vulnerability scanning pass.

## Tagging

After the preconditions pass, a maintainer selects the SemVer version, creates
and pushes a signed annotated tag at the verified revision, and publishes the
GitHub release with its reviewed notes and artifacts. Verify the public tag,
module proxy, checksum database, release assets, and package documentation
before reporting the version as available. These steps are not automated by
this repository today.
