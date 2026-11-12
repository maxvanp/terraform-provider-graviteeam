# v0.2.1 release preparation

Prepared on 2026-11-12. This note follows the v0.2.0 release record and
describes the changes that will ship as v0.2.1. It is not a publication
record. This change does not create a `v0.2.1` tag, a GitHub Release, or a
Terraform Registry version.

Publishing is a later push of tag `v0.2.1` on main. That push runs
[`.github/workflows/release.yml`](../.github/workflows/release.yml)
(GoReleaser). Previous releases were created by github-actions[bot].

The changes since v0.2.0 are already on main. The tip before this
preparation is `85b9b36b2b8966929b6626835c50da0c867b5f5d`, the merge of
[#49](https://github.com/maxvanp/terraform-provider-graviteeam/pull/49).
That pull request consolidates the unreleased changelog. The updates
below are in the history of that commit.

## Changes

Version **0.2.1** is a patch release. Provider resources are unchanged.

The compatibility target moves from Gravitee AM 4.13.2 to **4.13.3**.
The acceptance stack pins `graviteeio/am-management-api:4.13.3` and
`graviteeio/am-gateway:4.13.3` together (#37, #38). The bundled OpenAPI
reference is the 4.13.3 snapshot: still 213 paths and 275 schemas; only
the document version changed from 4.13.2.

The acceptance-test MongoDB image moves from `mongo:6.0` to `mongo:8.2`.
Dependabot ignores `mongo` versions >= 8.3. Gravitee AM 4.13 supports
MongoDB 4.4–8.2, so 8.2 is the top of that range. The MongoDB container
sets `GLIBC_TUNABLES=glibc.pthread.rseq=1` so 8.2 starts on GitHub-hosted
runners: they report kernel `7.0.0`, which trips the rseq startup check
(SERVER-121912), and Ubuntu's fix level is not visible in `uname -r`
(SERVER-136848). glibc then owns rseq and TCMalloc leaves its per-CPU
cache off.

Go moves from 1.26.10 to 1.27.4. The 1.27.3 step includes the
`crypto/x509` CA trust mitigation (CVE-2026-111943). Go 1.27.4 clears
`govulncheck ./...` for GO-2026-6859 / CVE-2026-97054. `golangci-lint`
moves from v2.11.4 to v2.15.0 so lint can typecheck Go 1.27.

Also on main since v0.2.0: `golang.org/x/oauth2` moves from 0.37.0 to
0.38.0 (#44), the acceptance JWKS nginx image moves from
`nginx:1.31.7-alpine` to `nginx:1.31.8-alpine` (#43), and
`goreleaser/goreleaser-action` moves from v7.2.3 to v7.3.0 (#27).
GoReleaser stays at v2.17.1. The local installation version
(`make install`) and the README usage example are 0.2.1.

## Validation

Nine checks on `85b9b36b2b8966929b6626835c50da0c867b5f5d` completed
successfully, including Acceptance Tests. These are the push workflows
for that commit. They are not a signed release run, and this preparation
commit is not part of that SHA.

| Check | Workflow | Evidence |
|-------|----------|----------|
| Workflow Lint | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523201/job/130532357260) |
| Lint | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523201/job/130532357276) |
| Build | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523201/job/130532356917) |
| Unit Tests | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523201/job/130532357408) |
| Acceptance Tests | Tests | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523201/job/130533369029) |
| Secret Scan | Security | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523177/job/130532361021) |
| Vulnerability Scan | Security | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523177/job/130532360607) |
| Verify Generated Docs | Documentation | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523281/job/130532357239) |
| Snapshot Artifacts | Release Check | [Successful job](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523250/job/130532359021) |

Workflow runs for that commit:

| Workflow | Evidence |
|----------|----------|
| Tests | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523201) |
| Security | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523177) |
| Documentation | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523281) |
| Release Check | [Successful run](https://github.com/maxvanp/terraform-provider-graviteeam/actions/runs/43023523250) |

Post-download signature checks and Terraform Registry installation were not
performed. The v0.1.1 publication record added those checks only after the
tag existed. The `v0.2.1` tag is not created here, so this file does not
name a released source commit.
