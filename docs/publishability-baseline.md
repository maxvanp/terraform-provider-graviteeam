# Publishability Baseline

This document records the release-readiness evidence for provider version `v0.1.0`. It is an evidence record, not authorization to publish or modify release credentials.

The initial baseline was merged into `main` through [PR #22](https://github.com/maxvanp/terraform-provider-graviteeam/pull/22) on 2026-07-15. The subsequent release-readiness changes were merged through [PR #25](https://github.com/maxvanp/terraform-provider-graviteeam/pull/25) on 2026-09-13.

## Scope

The released provider is a community provider at `registry.terraform.io/maxvanp/graviteeam`, built with Terraform Plugin Framework protocol version 6.

The compatibility target must be a precise Gravitee AM release, not an open-ended `4.x` claim. As of 2026-09-13, the supported target is Gravitee AM `4.12.6` because:

- `4.12.6` is the latest stable upstream Git tag; `4.13.0` is still alpha.
- Both `graviteeio/am-management-api:4.12.6` and `graviteeio/am-gateway:4.12.6` publish Linux AMD64 and ARM64 images.
- The upstream `4.12.6` Management API OpenAPI document is available.

The API audit and complete serial acceptance suite pass against those exact images.

## Status Vocabulary

| Status | Meaning |
|--------|---------|
| `PASS` | The evidence cited in the row proves the requirement for the reviewed implementation. |
| `FAIL` | Current evidence contradicts the requirement. |
| `PENDING` | Evidence is missing or the implementation is still changing. |
| `USER ACTION` | Publication-time account or secret ownership that cannot be completed safely by repository automation. |

## Baseline

| Area | Requirement | Evidence required | Current status |
|------|-------------|-------------------|----------------|
| Registry eligibility | The repository is owned by the publishing namespace, lowercase, named `terraform-provider-graviteeam`, and public. | [Published Registry provider](https://registry.terraform.io/providers/maxvanp/graviteeam/0.1.0); the public repository and namespace match. | `PASS` |
| Provider protocol | Production server and tests consistently use Terraform Plugin Protocol 6; the Registry manifest declares `6.0`. | `main.go`, acceptance factories, and valid `terraform-registry-manifest.json`. | `PASS` |
| Compatibility | README, generated provider docs, Compose, OpenAPI snapshot, tests, and changelog name the same exact Gravitee AM version. | Version-reference audit plus acceptance tests against the pinned images. | `PASS` |
| API coverage | Every OpenAPI family is classified as a resource, data source, read-like operation, operational action, alternate lifecycle, or intentional exclusion. No writable candidate is left unclassified. | `./scripts/audit-openapi-coverage.py --check-doc` passes against the target snapshot. | `PASS` |
| Resource semantics | Durable objects support correct create, read, update, delete, refresh, and import behavior where the API permits it. Remote deletion removes state; updates preserve unmanaged or relationship fields where required. | Acceptance tests, targeted unit tests, schema audits, and documented exceptions. | `PASS` |
| State contract | IDs are stable and documented; sensitive values are marked sensitive; optional/computed behavior is plan-safe; schema changes do not silently corrupt existing state. | Schema audit, provider schema validation, import tests, and upgrade review. | `PASS` |
| Public surface | Every registered resource and data source has implementation, registration, example, generated documentation, and appropriate tests. | Coverage audit reports zero missing artifacts and generated docs are clean. | `PASS` |
| Documentation | Registry index, README, examples, import syntax, compatibility, limitations, maintenance status, and security reporting are accurate for the released code. No installation path claims an unavailable artifact. | Generated-doc idempotence check, documentation review, and link/content audit. | `PASS` |
| Repository governance | License, contributing guide, code of conduct, security policy, issue templates, PR template, CODEOWNERS, changelog, and maintenance policy are present and internally consistent. | Repository file audit. | `PASS` |
| Dependencies | Go uses the current supported patch release; direct dependency versions are reviewed for compatibility; indirect upgrades are reviewed; module files are tidy and verified. | `go list -m -u all`, `go mod tidy`, and `go mod verify`. | `PASS` |
| Code quality | Formatting, lint, vet, build, unit tests, coverage audits, generated-doc checks, and repository diff checks pass. | `make verify-local`, `make lint`, `make vet`, and `make build`. | `PASS` |
| Security | The current Go vulnerability database reports no reachable vulnerabilities; no credentials are committed. The final GitHub security workflow result is tracked by the CI row. | Local `govulncheck ./...` plus Gitleaks history and working-tree scans pass after upgrading `google.golang.org/grpc` to `v1.82.1` for GO-2026-6061. | `PASS` |
| Acceptance | Terraform exercises all supported resource and data-source lifecycles against the exact pinned Gravitee AM images. Any unavailable commercial plugin is explicitly documented and independently covered as far as possible. | PR #25 passes the complete GitHub acceptance job against Gravitee AM 4.12.6 on the remediated commit. | `PASS` |
| Release artifacts | A signed v0.1.0 release produces correctly named ZIP archives, a protocol manifest checksum, SHA256 checksums, and a detached checksum signature. | [Published signed GitHub release](https://github.com/maxvanp/terraform-provider-graviteeam/releases/tag/v0.1.0) contains the six platform archives and signature assets. | `PASS` |
| Platforms | Release artifacts include at least Darwin AMD64/ARM64, Linux AMD64/ARM64/ARMv6, and Windows AMD64; Linux AMD64 is CGO-free and self-contained for HCP Terraform. | GoReleaser configuration and dry-run artifact inventory. | `PASS` |
| Versioning | The first release version and changelog accurately describe compatibility and breaking-change expectations; released artifacts are immutable. | Changelog and release workflow review. | `PASS` |
| CI | Tests, workflow lint, lint, build, acceptance, generated docs, vulnerability scans, secret scans, and release snapshot validation pass for the exact release source. | PR #25's release source `5ee7b28` passed all 9 checks, including Documentation, Release Check, Security, and Tests. | `PASS` |
| Change review | The release-readiness work is merged with no accidental files, unresolved conflicts, or unexplained generated changes. | The security, release-hardening, and AM compatibility changes were reviewed in PR #25 and merged into `main` on 2026-09-13. | `PASS` |
| Final report | Every readiness row is updated with direct evidence for the published release. | This report, the PR #25 check suite, and the published release and Registry records contain the evidence. | `PASS` |

## Final Evidence

| Requirement group | Result |
|-------------------|--------|
| Supported API | Gravitee AM `4.12.6` is pinned in Compose, documentation, changelog, and the bundled upstream OpenAPI snapshot; the complete acceptance suite passes against its exact images. |
| API classification | `205` paths form `135` audited families. All `12` uncovered writable families are classified; there are `0` unclassified durable candidates and `0` uncovered read-only families. |
| Provider surface | `53` resources and `20` data sources are registered. All `73` types have test, example, generated-documentation, and client/API coverage artifacts. |
| Resource and state semantics | All `53` import-capable resources have import tests. CRUD, missing-object removal, update preservation, drift, sensitive state, stable IDs, and schema behavior are covered by acceptance and targeted tests. |
| Test evidence | `73/73` Terraform types have acceptance artifacts, `72/73` execute against stock Compose, and the sole exclusion is the documented Enterprise/technical-preview OpenFGA engine. Total statement coverage is `97.4%`. |
| Quality and security | Formatting, lint, vet, build, module verification, unit tests, schema/API audits, generated docs, `govulncheck`, Actionlint, and full-history plus working-tree Gitleaks scans pass. |
| Release engineering | GoReleaser validates and publishes six ZIP archives with signed checksums for Darwin AMD64/ARM64, Linux AMD64/ARM64/ARMv6, and Windows AMD64. Archive hashes and the Registry manifest hash match the generated SHA256SUMS. |
| Repository and CI | PR #25 release source `5ee7b28` passed all 9 checks after remediating GO-2026-6061; the acceptance job passed against Gravitee AM 4.12.6, and direct Terraform Registry installation of `v0.1.0` succeeded with checksum/signature verification. |

Publication and installation verification of `v0.1.0` are complete.

## Required Verification

Run these commands from a clean checkout to reproduce the release and artifact verification:

Install GoReleaser `v2.17.1` on `PATH` to match both release workflows.
`make release-check` also tests that missing or corrupt archive checksums are rejected.

```bash
make fmt
make lint
make vet
make build
make verify-local
go mod verify
govulncheck ./...
docker compose -f docker-compose.test.yml up -d
make local-gap-probe
TF_ACC=1 make testacc
docker compose -f docker-compose.test.yml down --remove-orphans
make release-check
gitleaks git --no-banner --redact --verbose .
gitleaks dir --no-banner --redact --verbose .
git diff --check
git status --short
```

The GitHub `Tests`, `Documentation`, and `Security` workflows passed for release source `5ee7b28`. A local pass does not substitute for CI, and CI does not substitute for the local release-artifact inspection. Snapshot checks use `--skip=sign` so routine validation does not require access to signing credentials; the release workflow signs the published checksums.

## Publication Record

1. [GitHub release `v0.1.0`](https://github.com/maxvanp/terraform-provider-graviteeam/releases/tag/v0.1.0) is signed and published from release source `5ee7b28`.
2. [Terraform Registry `maxvanp/graviteeam` `v0.1.0`](https://registry.terraform.io/providers/maxvanp/graviteeam/0.1.0) is published with six platforms and protocol 6.
3. Direct Terraform 1.14.7 installation and validation succeeded with 53 resources and 20 data sources.
4. Public OpenPGP fingerprint: `96BE5305AEB21B9D1877B2E40C698A4920F58A9A`.

The private signing material remains outside the repository and is provided to the release workflow through GitHub secrets.

## Primary References

- [HashiCorp: Publish providers](https://developer.hashicorp.com/terraform/registry/providers/publishing)
- [HashiCorp: Provider documentation](https://developer.hashicorp.com/terraform/registry/providers/docs)
- [HashiCorp: Recommended operating systems and architectures](https://developer.hashicorp.com/terraform/registry/providers/os-arch)
- [HashiCorp: Provider acceptance tests](https://developer.hashicorp.com/terraform/plugin/framework/acctests)
- [HashiCorp: Provider best practices](https://developer.hashicorp.com/terraform/plugin/best-practices)
- [Gravitee AM `4.12.6` source tag](https://github.com/gravitee-io/gravitee-access-management/tree/4.12.6)
- [Go releases](https://go.dev/doc/devel/release)
