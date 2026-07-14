# Security policy

## Supported versions

Before `v1.0.0`, security fixes are applied to the latest revision of `main`.
After the first stable release, the project will document supported release
lines here and provide fixes for the latest stable minor line.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting for this repository. Include:

- the affected parser or ingest surface;
- a minimal reproducer or malformed input file;
- expected and observed behavior;
- potential availability, integrity, or confidentiality impact;
- any suggested mitigation.

Do not include secrets or production data. Please allow maintainers reasonable
time to confirm and coordinate a fix before public disclosure.

## Security posture

The package processes untrusted tabular inputs, archives, encodings, and row
data. Its CI should therefore include static analysis, dependency
vulnerability scanning, fixture regressions, and format-specific hardening
tests. Applications should still apply request, file-size, archive, and memory
limits before handing inputs to the package.

Delimiter parsing, spreadsheet handling, fixed-width extraction, encoding
conversion, and archive extraction should be treated as hostile-input
boundaries. Public documentation must explain known limits and unsupported
cases clearly rather than implying broader guarantees than the code provides.
