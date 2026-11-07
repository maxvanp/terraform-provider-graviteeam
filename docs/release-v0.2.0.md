# v0.2.0 release preparation

Prepared on 2026-11-07. This note follows the v0.1.1 release record and
describes the changes that will ship as v0.2.0. It is not a publication
record. This change does not create a `v0.2.0` tag, a GitHub Release, or a
Terraform Registry version.

Publishing is a later push of tag `v0.2.0` on main. That push runs
[`.github/workflows/release.yml`](../.github/workflows/release.yml)
(GoReleaser). Previous releases were created by github-actions[bot].

The provider changes are on main in
[#35](https://github.com/maxvanp/terraform-provider-graviteeam/pull/35),
merge commit `f05187055572743bf7389fda5abf83868aaa51aa`.

## Changes

Version **0.2.0** is a minor bump, not 0.1.2. The compatibility target
moves from Gravitee AM 4.12.7 to **4.13.2**, and trust-domain behavior
changes.

The acceptance stack pins `graviteeio/am-management-api:4.13.2` and
`graviteeio/am-gateway:4.13.2` together. The Gateway pin landed first in
#34. The bundled OpenAPI reference is the 4.13.2 snapshot: 213 paths and
275 schemas, up from 205 paths and 249 schemas in 4.12.7.

`graviteeam_trust_domain` calls `/trusted-domains`. The resource name is
the trusted-domain label and the SPIFFE trust domain, and the JWKS URL is
sent as `keyMaterial`. Domain updates keep `keyRetrievalSettings`, and
`data_plane_id` cannot change after creation on AM 4.13. Reporter updates
keep `attributeMappings` and `attributeMappingEventTypes`. Trust-domain
acceptance sends the domain JWKS policy through `keyRetrievalSettings`,
which is where AM 4.13 enforces HTTP and private-address checks.

Documented AM 4.13 gaps stay unmanaged: deprecated `/trust-domains`,
protected-resource flows, current-user preferences, and the new read-only
license, installation, cursor, and Cross App Access catalogs.

Go is 1.26.10 and `golang.org/x/net` is v0.62.0. The changelog records a
clean `govulncheck ./...` for GO-2026-6809, GO-2026-6617, GO-2026-6613,
GO-2026-6612, GO-2026-6611, GO-2026-6610, GO-2026-6608, GO-2026-6607,
GO-2026-6605, GO-2026-6603, GO-2026-6866, and GO-2026-6860.
`golang.org/x/oauth2` moves from 0.36.0 to 0.37.0 (#26). The local
installation version (`make install`) and the README usage example are
0.2.0.

## Validation

Nine checks on the #35 merge commit
`f05187055572743bf7389fda5abf83868aaa51aa` completed successfully, including
Acceptance Tests. These are the push workflows for that commit. They are
not a signed release run, and this preparation commit is not part of that
SHA.

| Check | Workflow | Evidence |
|-------|----------|----------|
| Workflow Lint | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603546/job/128183887152) |
| Lint | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603546/job/128183887366) |
| Build | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603546/job/128183887346) |
| Unit Tests | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603546/job/128183886903) |
| Acceptance Tests | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603546/job/128184828774) |
| Secret Scan | Security | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603435/job/128183886625) |
| Vulnerability Scan | Security | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603435/job/128183886368) |
| Verify Generated Docs | Documentation | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603548/job/128183886964) |
| Snapshot Artifacts | Release Check | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603432/job/128183886566) |

Workflow runs for that commit:

| Workflow | Evidence |
|----------|----------|
| Tests | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603546) |
| Security | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603435) |
| Documentation | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603548) |
| Release Check | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/42302603432) |

Post-download signature checks and Terraform Registry installation were not
performed. The v0.1.1 publication record added those checks only after the
tag existed. The `v0.2.0` tag is not created here, so this file does not
name a released source commit.
