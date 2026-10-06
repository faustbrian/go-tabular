# Security Policy

## Supported Versions

Security fixes are applied to the latest stable v1 release and `main`.
Additional supported release lines and end-of-support dates will be documented
here when offered.

The finite-default hardening in the active source changes documented behavior
and is therefore planned for unpublished v2. Until v2 is published, v1 callers
processing untrusted input must set explicit positive limits and enforce
transport and job-level resource controls.

## Reporting A Vulnerability

Use [GitHub private vulnerability reporting](https://github.com/faustbrian/go-tabular/security/advisories/new)
for this repository. Include a minimal reproducer, expected and observed
behavior, affected versions, impact, and any suggested mitigation. Do not
include secrets or production data.

## Response Process

Maintainers will acknowledge the report, reproduce and assess it privately,
coordinate a fix and advisory, and credit the reporter when requested. Public
disclosure should wait until a fix or agreed mitigation is available.

## Package Security Boundary

Files, archives, workbooks, encodings, records, headers, formulas, and row values are untrusted ingest inputs. Archive, allocation, record, and encoding limits are part of the maintained security boundary.

## Application Responsibilities

Applications remain responsible for transport limits, authentication,
authorization, rate limiting, deadlines, secret handling, deployment policy,
and business-level validation. Package safeguards do not replace those
controls.

See [docs/security.md](docs/security.md) and
[docs/behavior-and-limits.md](docs/behavior-and-limits.md) for adoption
guidance and maintained input boundaries.
