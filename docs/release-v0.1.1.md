# v0.1.1 publication record

Published on 2026-10-01 from source commit
`361f7433aaa5682e1d9f104fdb35eac92c7d9a5d`:

- [GitHub release v0.1.1](https://github.com/maxvanp/terraform-provider-graviteeam/releases/tag/v0.1.1)
- [Terraform Registry v0.1.1](https://registry.terraform.io/providers/maxvanp/graviteeam/0.1.1)
- [Successful signed-release workflow](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/36899010195)

## Changes

The compatibility target and acceptance stack are Gravitee AM **4.12.7**.
The upstream Management API retains the same 205 paths and 249 schemas as
4.12.6. This patch adds no resources or data sources and makes no provider
schema changes. AM 4.13 prerelease features are outside this release.

gRPC 1.83.2 remediates reachable vulnerabilities GO-2026-6443 and
GO-2026-6348. Trust-domain acceptance tests use a local public JWKS fixture
instead of depending on external DNS and a third-party endpoint.

## Validation

All four required workflows passed for the exact released source:

| Workflow | Evidence |
|----------|----------|
| Tests | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/36897937027) |
| Security | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/36897936969) |
| Documentation | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/36897936972) |
| Release Check | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/36897936971) |

Local verification also passed formatting, lint, vet, build, module integrity,
unit tests, generated documentation, API/schema/coverage audits, workflow
lint, secret scans, and release snapshot checks. Statement coverage is
**97.4%**. The complete serial acceptance suite passed **70 tests** against
the pinned AM 4.12.7 stack; one Enterprise OpenFGA test was skipped because
the stock images do not provide that commercial plugin.

The GitHub Security workflow passed its live vulnerability scan. Local
`govulncheck` reported no vulnerabilities using a checkout of the official
Go vulnerability database at revision `5775cfa` (2026-09-28), because this
environment could not reach `vuln.go.dev`.

## Published artifact verification

The published detached SHA256SUMS signature was verified with the repository's
[release public key](release-signing-key.asc), pinned to fingerprint:

```text
96BE5305AEB21B9D1877B2E40C698A4920F58A9A
```

All six downloaded ZIP archives and the protocol manifest matched the signed
checksums. Each ZIP contained the correctly named provider binary. Platforms
are Darwin AMD64/ARM64, Linux AMD64/ARM64/ARMv6, and Windows AMD64. The
published manifest matches the source manifest and declares protocol `6.0`.

Terraform **1.15.8** installed `maxvanp/graviteeam` **0.1.1** directly from the
Registry with signature verification and no development override. Configuration
validation passed, and its schema exposed **53 resources and 20 data sources**.
The installed binary created, refreshed, and destroyed a temporary domain on
the local AM 4.12.7 stack; the resulting Terraform state contained no managed
resources.

The release tag remains on the source commit above. This publication record
is added after release and does not change the released artifacts.
