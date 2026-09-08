#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
required=(
  README.md CHANGELOG.md LICENSE SECURITY.md SUPPORT.md CONTRIBUTING.md
  CODE_OF_CONDUCT.md COMPATIBILITY.md ROADMAP.md docs/README.md
  docs/quickstart.md docs/adoption.md docs/api.md docs/architecture.md
  docs/behavior-and-limits.md docs/compatibility.md docs/cookbook.md
  docs/examples.md docs/faq.md docs/formats.md
  docs/go-safety-and-concurrency.md docs/migration.md docs/performance.md
  docs/releasing.md docs/repository-standards.md docs/security.md
  docs/specification-decisions.md docs/troubleshooting.md
)

cd "${root}"
for path in "${required[@]}"; do
  test -s "${path}" || {
    printf 'required documentation is missing or empty: %s\n' "${path}" >&2
    exit 1
  }
done

go test ./... -run '^Example' -count=1
go vet ./...
