# Publishability Baseline

This document defines the evidence required before this provider is considered ready for a first public release. It is a release-readiness contract, not authorization to publish, tag, merge, or create a GitHub release.

## Scope

The intended first release is a community provider at `registry.terraform.io/maxvanp/graviteeam`, built with Terraform Plugin Framework protocol version 6.

The compatibility target must be a precise Gravitee AM release, not an open-ended `4.x` claim. As of 2026-07-14, the supported target is Gravitee AM `4.12.1` because:

- `4.12.1` is the latest stable upstream Git tag; `4.13.0` is still alpha.
- Both `graviteeio/am-management-api:4.12.1` and `graviteeio/am-gateway:4.12.1` publish Linux AMD64 and ARM64 images.
- The upstream `4.12.1` Management API OpenAPI document is available.

The API audit and complete serial acceptance suite pass against those exact images.

## Status Vocabulary

| Status | Meaning |
|--------|---------|
| `PASS` | Current evidence proves the requirement for the exact candidate commit. |
| `FAIL` | Current evidence contradicts the requirement. |
| `PENDING` | Evidence is missing or the implementation is still changing. |
| `USER ACTION` | Publication-time account or secret ownership that cannot be completed safely by repository automation. |

## Baseline

| Area | Requirement | Evidence required | Current status |
|------|-------------|-------------------|----------------|
| Registry eligibility | The repository is owned by the publishing namespace, lowercase, named `terraform-provider-graviteeam`, and public before Registry onboarding. | GitHub repository metadata and Terraform Registry naming rules. The name and ownership pass; visibility is currently private. | `USER ACTION` |
| Provider protocol | Production server and tests consistently use Terraform Plugin Protocol 6; the Registry manifest declares `6.0`. | `main.go`, acceptance factories, and valid `terraform-registry-manifest.json`. | `PASS` |
| Compatibility | README, generated provider docs, Compose, OpenAPI snapshot, tests, and changelog name the same exact Gravitee AM version. | Version-reference audit plus acceptance tests against the pinned images. | `PASS` |
| API coverage | Every OpenAPI family is classified as a resource, data source, read-like operation, operational action, alternate lifecycle, or intentional exclusion. No writable candidate is left unclassified. | `./scripts/audit-openapi-coverage.py --check-doc` passes against the target snapshot. | `PASS` |
| Resource semantics | Durable objects support correct create, read, update, delete, refresh, and import behavior where the API permits it. Remote deletion removes state; updates preserve unmanaged or relationship fields where required. | Acceptance tests, targeted unit tests, schema audits, and documented exceptions. | `PASS` |
| State contract | IDs are stable and documented; sensitive values are marked sensitive; optional/computed behavior is plan-safe; schema changes do not silently corrupt existing state. | Schema audit, provider schema validation, import tests, and upgrade review. | `PASS` |
| Public surface | Every registered resource and data source has implementation, registration, example, generated documentation, and appropriate tests. | Coverage audit reports zero missing artifacts and generated docs are clean. | `PASS` |
| Documentation | Registry index, README, examples, import syntax, compatibility, limitations, maintenance status, and security reporting are accurate for the final code. No installation path claims an artifact that does not exist yet. | Generated-doc idempotence check, documentation review, and link/content audit. | `PASS` |
| Repository governance | License, contributing guide, code of conduct, security policy, issue templates, PR template, CODEOWNERS, changelog, and maintenance policy are present and internally consistent. | Repository file audit. | `PASS` |
| Dependencies | Go uses the current supported patch release; direct dependencies are current; indirect upgrades are reviewed; module files are tidy and verified. | `go list -m -u all`, `go mod tidy`, and `go mod verify`. | `PASS` |
| Code quality | Formatting, lint, vet, build, unit tests, coverage audits, generated-doc checks, and repository diff checks pass. | `make verify-local`, `make lint`, `make vet`, and `make build`. | `PASS` |
| Security | The current Go vulnerability database reports no reachable vulnerabilities; no credentials are committed. The final GitHub security workflow result is tracked by the CI row. | `govulncheck ./...` plus Gitleaks history and working-tree scans. | `PASS` |
| Acceptance | Terraform exercises all supported resource and data-source lifecycles against the exact pinned Gravitee AM images. Any unavailable commercial plugin is explicitly documented and independently covered as far as possible. | Complete local Docker acceptance run; the final GitHub result is tracked by the CI row. | `PASS` |
| Release artifacts | A clean snapshot build produces correctly named ZIP archives, a protocol manifest checksum, SHA256 checksums, and detached checksum signature configuration. | `goreleaser check`, successful unsigned snapshot, and artifact/hash inspection without publishing. | `PASS` |
| Platforms | Release artifacts include at least Darwin AMD64/ARM64, Linux AMD64/ARM64/ARMv6, and Windows AMD64; Linux AMD64 is CGO-free and self-contained for HCP Terraform. | GoReleaser configuration and dry-run artifact inventory. | `PASS` |
| Versioning | The first release version and changelog accurately describe compatibility and breaking-change expectations; released artifacts are immutable. | Changelog and release workflow review. | `PASS` |
| CI | Tests, workflow lint, lint, build, acceptance, generated docs, and vulnerability scans pass for the exact final pull-request SHA. | Green GitHub checks associated with the final commit. | `PENDING` |
| Change review | The unpublished work is preserved on a review branch with no accidental files, unresolved conflicts, or unexplained generated changes. | Clean worktree, reviewed diff, intentional commits, and pull request. | `PENDING` |
| Final report | Every row in this table is updated with direct evidence and no `FAIL` or `PENDING` remains. Publication-only prerequisites are listed separately. | Final readiness report committed on the candidate branch. | `PENDING` |

## Required Verification

Run these commands from a clean checkout of the final candidate commit:

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
goreleaser check
goreleaser release --snapshot --clean --skip=sign
gitleaks git --no-banner --redact --verbose .
gitleaks dir --no-banner --redact --verbose .
git diff --check
git status --short
```

The GitHub `Tests`, `Documentation`, and `Security` workflows must then pass for the exact final SHA. A local pass does not substitute for CI, and CI does not substitute for the local release-artifact inspection. The unsigned snapshot uses `--skip=sign` because the maintainer-owned release key is deliberately unavailable locally; `goreleaser check` still validates the detached checksum signature configuration.

## Publication-Only User Actions

These actions are deliberately outside the readiness implementation and must not be performed until publication is discussed:

1. Choose or generate a dedicated RSA or DSA GPG signing key and retain its private key securely.
2. Add the armored public key to the `maxvanp` Terraform Registry namespace.
3. Configure the GitHub `GPG_PRIVATE_KEY` and `PASSPHRASE` repository secrets.
4. Change the reviewed repository visibility from private to public.
5. Sign in to the Terraform Registry with the GitHub account that owns the public repository and accept the Registry terms.
6. Approve the final version number, merge, tag, GitHub release, and Registry publication.

Readiness requires proving that the repository consumes these inputs correctly. It does not require exposing, generating, or uploading the maintainer's private signing material.

## Primary References

- [HashiCorp: Publish providers](https://developer.hashicorp.com/terraform/registry/providers/publishing)
- [HashiCorp: Provider documentation](https://developer.hashicorp.com/terraform/registry/providers/docs)
- [HashiCorp: Recommended operating systems and architectures](https://developer.hashicorp.com/terraform/registry/providers/os-arch)
- [HashiCorp: Provider acceptance tests](https://developer.hashicorp.com/terraform/plugin/framework/acctests)
- [HashiCorp: Provider best practices](https://developer.hashicorp.com/terraform/plugin/best-practices)
- [Gravitee AM `4.12.1` source tag](https://github.com/gravitee-io/gravitee-access-management/tree/4.12.1)
- [Go releases](https://go.dev/doc/devel/release)
