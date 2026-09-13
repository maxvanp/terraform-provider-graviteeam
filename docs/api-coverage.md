# Gravitee AM API Coverage

This document tracks the provider coverage against the bundled Gravitee AM Management API reference.

## Source

| Item | Value |
|------|-------|
| Target Gravitee AM version | `4.12.6` |
| Bundled OpenAPI file | [`docs/openapi.yaml`](openapi.yaml) |
| Bundled OpenAPI version | `4.12.6` |
| OpenAPI path entries | `205` |
| Path entries with at least one write verb | `126` |
| Terraform resources registered | `53` |
| Terraform data sources registered | `20` |

The [AM 4.12 changelog](https://documentation.gravitee.io/am/releases-and-changelog/changelog/am-4.12.x) lists 4.12.2–4.12.6 as bug-fix releases, including user deletion, permission caching, and console login corrections. The upstream 4.12.6 snapshot has no path or schema changes from 4.12.1, so this update changes the tested server version without changing Terraform resource schemas.

The OpenAPI file is a reference snapshot only. Refresh it from the official Gravitee repository with:

```bash
./scripts/update-openapi.sh
```

Run the coverage audit with:

```bash
./scripts/audit-openapi-coverage.py
```

When the local docker-compose stack is running, verify plugin availability with:

```bash
./scripts/probe-local-compose-plugins.py --check
```

When the local docker-compose stack is running, verify documented writable non-resource gap behavior with:

```bash
make local-gap-probe
```

Current automated audit summary:

| Item | Value |
|------|-------|
| OpenAPI families | `135` |
| Writable families | `79` |
| Read-only families | `56` |
| Uncovered writable families without Terraform resource | `12` |
| Unclassified writable resource candidates | `0` |
| Writable families covered only by data source | `7` |
| Uncovered read-only families | `0` |
| Registered resources missing test/doc/example artifact | `0` |
| Registered data sources missing test/doc/example artifact | `0` |

Coverage below is grouped by API family, not by every individual OpenAPI path. Some API paths are action-only or operational endpoints and are not good Terraform resource candidates. Writable families require a Terraform resource to count as covered; a read-only data source is useful, but it does not make the writable API family fully managed.

Current test coverage baseline:

| Item | Value |
|------|-------|
| Terraform types with acceptance coverage | `73/73` |
| Terraform types executable in stock local compose acceptance | `72/73` |
| Import-capable resources with import tests | `53/53` |
| Resources with disabled import verification | `0` |
| `go test ./... -coverprofile=/tmp/graviteeam-coverage.out -covermode=atomic` | `98.1%` total statement coverage |

The remaining local compose acceptance gap is `graviteeam_authorization_engine`. In the Gravitee AM 4.12.1 baseline probe, the image loaded the `openfga` authorization engine zip, but the local API reported it as `deployed=false` with `feature=am-authorizationengine-openfga`, and its platform schema endpoint did not return a usable schema body. Gravitee documents this OpenFGA authorization engine as a technical preview that requires access from Gravitee, and the plugin marketplace marks it as Enterprise.

## Covered Resources

| API family | Terraform resource |
|------------|--------------------|
| `domain:alerts/notifiers` | `graviteeam_alert_notifier` |
| `domain:alerts/triggers` | `graviteeam_alert_trigger` |
| `domain:applications` | `graviteeam_application` |
| `domain:applications/emails` | `graviteeam_application_email` |
| `domain:applications/flows` | `graviteeam_application_flow` |
| `domain:applications/forms` | `graviteeam_application_form` |
| `domain:applications/members` | `graviteeam_application_member` |
| `domain:applications/secrets` | `graviteeam_application_secret` |
| `domain:applications/secrets/_renew` | `graviteeam_application_secret` |
| `domain:applications/type` | `graviteeam_application` |
| `domain:authorization-engines` | `graviteeam_authorization_engine` |
| `domain:auth-device-notifiers` | `graviteeam_auth_device_notifier` |
| `domain:bot-detections` | `graviteeam_bot_detection` |
| `domain:certificates` | `graviteeam_certificate` |
| `domain:certificates/rotate` | `graviteeam_generated_certificate` |
| `domain:certificate-settings` | `graviteeam_domain_certificate_settings` |
| `domain:device-identifiers` | `graviteeam_device_identifier` |
| `domain:emails` | `graviteeam_email_template` |
| `domain:extensionGrants` | `graviteeam_extension_grant` |
| `domain:factors` | `graviteeam_factor` |
| `domain:flows` | `graviteeam_domain_flow` |
| `domain:forms` | `graviteeam_form` |
| `domain:groups` | `graviteeam_group` |
| `domain:groups/members` | `graviteeam_group_members` |
| `domain:groups/roles` | `graviteeam_group_roles` |
| `domain:i18n/dictionaries` | `graviteeam_i18n_dictionary` |
| `domain:i18n/dictionaries/entries` | `graviteeam_i18n_dictionary` |
| `domain:identities` | `graviteeam_identity_provider` |
| `domain:identities/password-policy` | `graviteeam_identity_provider_password_policy` |
| `domain:members` | `graviteeam_domain_member` |
| `domain:password-policies` | `graviteeam_password_policy` |
| `domain:password-policies/default` | `graviteeam_password_policy` |
| `domain:protected-resources` | `graviteeam_protected_resource` |
| `domain:protected-resources/members` | `graviteeam_protected_resource_member` |
| `domain:protected-resources/secrets` | `graviteeam_protected_resource_secret` |
| `domain:protected-resources/secrets/_renew` | `graviteeam_protected_resource_secret` |
| `domain:reporters` | `graviteeam_reporter` |
| `domain:resources` | `graviteeam_service_resource` |
| `domain:roles` | `graviteeam_role` |
| `domain:scopes` | `graviteeam_scope` |
| `domain:themes` | `graviteeam_theme` |
| `domain:trust-domains` | `graviteeam_trust_domain` |
| `domain:users` | `graviteeam_user` |
| `domain:users/cert-credentials` | `graviteeam_user_certificate_credential` |
| `domain:users/lock` | `graviteeam_user` |
| `domain:users/resetPassword` | `graviteeam_user` |
| `domain:users/roles` | `graviteeam_user_role` |
| `domain:users/sendRegistrationConfirmation` | `graviteeam_user` |
| `domain:users/status` | `graviteeam_user` |
| `domain:users/unlock` | `graviteeam_user` |
| `domain:users/username` | `graviteeam_user` |
| `environment:domains` | `graviteeam_domain` |
| `org:entrypoints` | `graviteeam_org_entrypoint` |
| `org:forms` | `graviteeam_org_form` |
| `org:groups` | `graviteeam_org_group` |
| `org:groups/members` | `graviteeam_org_group_members` |
| `org:identities` | `graviteeam_org_identity_provider` |
| `org:members` | `graviteeam_org_member` |
| `org:reporters` | `graviteeam_org_reporter` |
| `org:roles` | `graviteeam_org_role` |
| `org:settings` | `graviteeam_org_settings` |
| `org:tags` | `graviteeam_org_tag` |
| `org:users` | `graviteeam_org_user` |
| `org:users/resetPassword` | `graviteeam_org_user` |
| `org:users/status` | `graviteeam_org_user` |
| `org:users/tokens` | `graviteeam_org_user_token` |
| `org:users/username` | `graviteeam_org_user` |

## Covered Data Sources

| API family | Terraform data source |
|------------|-----------------------|
| `domain:users/audits` | `graviteeam_admin_metadata` |
| `domain:analytics` | `graviteeam_analytics` |
| `domain:applications/analytics` | `graviteeam_application_metadata` |
| `domain:applications/members/permissions` | `graviteeam_permissions_metadata` |
| `domain:applications/resources` | `graviteeam_application_metadata` |
| `domain:applications/resources/policies` | `graviteeam_application_metadata` |
| `domain:applications/search` | `graviteeam_applications` |
| `domain:applications/search/_cursor` | `graviteeam_applications` |
| `domain:audits` | `graviteeam_audits` |
| `domain:certificates/key` | `graviteeam_domain_metadata` |
| `domain:certificates/keys` | `graviteeam_domain_metadata` |
| `domain:entrypoints` | `graviteeam_entrypoints` |
| `domain:flows` | `graviteeam_flows` |
| `domain:forms/preview` | `graviteeam_form_preview` |
| `domain:members/permissions` | `graviteeam_permissions_metadata` |
| `domain:password-policies/activePolicy` | `graviteeam_domain_metadata` |
| `domain:password-policies/evaluate` | `graviteeam_password_policy_evaluation` |
| `domain:protected-resources/members/permissions` | `graviteeam_permissions_metadata` |
| `environment:data-planes` | `graviteeam_environment_metadata` |
| `environment:data-sources` | `graviteeam_environment_metadata` |
| `environment:domains/_hrid` | `graviteeam_admin_metadata` |
| `environment:members/permissions` | `graviteeam_permissions_metadata` |
| `org:audits` | `graviteeam_admin_metadata` |
| `org:environments` | `graviteeam_admin_metadata` |
| `platform:audits/events` | `graviteeam_platform_metadata` |
| `platform:configuration/alerts/status` | `graviteeam_platform_metadata` |
| `platform:configuration/flow/schema` | `graviteeam_platform_metadata` |
| `platform:configuration/spel/grammar` | `graviteeam_platform_metadata` |
| `platform:configuration/users/email-required` | `graviteeam_platform_metadata` |
| `platform:installation` | `graviteeam_platform_metadata` |
| `platform:license` | `graviteeam_platform_metadata` |
| `platform:plugins/auth-device-notifiers` | `graviteeam_plugins` |
| `platform:plugins/auth-device-notifiers/schema` | `graviteeam_plugins` |
| `platform:plugins/authorization-engines` | `graviteeam_plugins` |
| `platform:plugins/authorization-engines/schema` | `graviteeam_plugins` |
| `platform:plugins/bot-detections` | `graviteeam_plugins` |
| `platform:plugins/bot-detections/schema` | `graviteeam_plugins` |
| `platform:plugins/certificates` | `graviteeam_plugins` |
| `platform:plugins/certificates/schema` | `graviteeam_plugins` |
| `platform:plugins/device-identifiers` | `graviteeam_plugins` |
| `platform:plugins/device-identifiers/schema` | `graviteeam_plugins` |
| `platform:plugins/extensionGrants` | `graviteeam_plugins` |
| `platform:plugins/extensionGrants/schema` | `graviteeam_plugins` |
| `platform:plugins/factors` | `graviteeam_plugins` |
| `platform:plugins/factors/schema` | `graviteeam_plugins` |
| `platform:plugins/identities` | `graviteeam_plugins` |
| `platform:plugins/identities/schema` | `graviteeam_plugins` |
| `platform:plugins/notifiers` | `graviteeam_plugins` |
| `platform:plugins/notifiers/schema` | `graviteeam_plugins` |
| `platform:plugins/policies` | `graviteeam_plugins` |
| `platform:plugins/policies/documentation` | `graviteeam_plugins` |
| `platform:plugins/policies/schema` | `graviteeam_plugins` |
| `platform:plugins/reporters` | `graviteeam_plugins` |
| `platform:plugins/reporters/schema` | `graviteeam_plugins` |
| `platform:plugins/resources` | `graviteeam_plugins` |
| `platform:plugins/resources/schema` | `graviteeam_plugins` |
| `platform:roles` | `graviteeam_platform_metadata` |
| `self:_root` | `graviteeam_self_metadata` |
| `self:notifications` | `graviteeam_self_metadata` |
| `domain:users/consents` | `graviteeam_user_consents` |
| `domain:users/credentials` | `graviteeam_user_credentials` |
| `domain:users/devices` | `graviteeam_user_devices` |
| `domain:users/factors` | `graviteeam_user_factors` |
| `domain:users/identities` | `graviteeam_user_identities` |

## Known Gaps

These API families expose write operations in the OpenAPI reference but are not currently modeled as Terraform resources.

### High-Value Resource Candidates

None currently identified in the 4.12.6 OpenAPI audit. Compared with 4.12.1, all 205 paths and 249 schemas are unchanged; only `info.version` differs in the upstream snapshot.

### Read-Like POST Endpoints Not Covered

These endpoints look closer to calculated reads than durable Terraform resources. They are exposed as data sources, but remain listed here because the coverage audit only counts Terraform resources as full writable-family coverage.

| API family | Notes |
|------------|-------|
| `domain:forms/preview` | Template preview operation exposed by `graviteeam_form_preview`; the local API requires lower-case template names for preview even though CRUD form resources use upper-case template names. |
| `domain:password-policies/evaluate` | Password policy evaluation operation exposed by `graviteeam_password_policy_evaluation`; local 4.12.1 accepts concrete policy IDs. |

### Action or Lifecycle Endpoints

These endpoints may be better represented as explicit resources, one-shot actions, or left unmanaged depending on Terraform semantics.

| API family | Notes |
|------------|-------|
| `domain:users/bulk` | Reachable in local gap probe, but this is a batch alternative to `graviteeam_user` create/update/delete rather than a distinct durable object. |
| `domain:cimd/applications` | CIMD creates an application from external metadata rather than a distinct durable object; it is an alternate application lifecycle operation. |
| `domain:cimd/validate` | CIMD validation is an operational preview/validation request and has no durable Terraform state. |
| `domain:users/consents` | User consent lifecycle; read-only state is exposed by `graviteeam_user_consents`, revocation remains unmanaged until a consent fixture can prove safe desired-state semantics; local gap probe covers both collection delete by `clientId` and item revoke behavior. |
| `domain:users/credentials` | User credential lifecycle; read-only state is exposed by `graviteeam_user_credentials`, revocation remains unmanaged until a credential fixture can prove safe desired-state semantics. |
| `domain:users/devices` | User device lifecycle; read-only state is exposed by `graviteeam_user_devices`, deletion remains unmanaged until a device fixture can prove safe desired-state semantics. |
| `domain:users/factors` | User factor lifecycle; read-only state is exposed by `graviteeam_user_factors`, revocation remains unmanaged; local gap probe returned `204` for a missing factor id. |
| `domain:users/identities` | User identity lifecycle; read-only state is exposed by `graviteeam_user_identities`, unlink remains unmanaged; local gap probe returned `204` for a missing identity id. |
| `org:users/bulk` | Reachable in local gap probe, but this is a batch alternative to `graviteeam_org_user` create/update/delete rather than a distinct durable object. |
| `self:notifications/acknowledge` | Current-user notification acknowledgement operation; reachable in local gap probe but scoped to ephemeral current-user notification state. |

### Read-Only and Admin Metadata Not Covered

The provider currently exposes all read-only and platform metadata families identified by the audit.

| API family | Notes |
|------------|-------|
| None | No uncovered read-only families remain in the bundled OpenAPI snapshot. |

## Suggested Implementation Order

1. Revisit action-only endpoints and decide case by case whether Terraform should model them.
2. Add read-only data sources for useful runtime and platform metadata families.
3. Reassess local compose plugins when plugin-backed resources are skipped because plugins are unavailable.

## Verification

After changing API coverage, run:

```bash
./scripts/audit-openapi-coverage.py
./scripts/audit-openapi-coverage.py --check-doc
./scripts/audit-test-coverage.py --check
./scripts/probe-openapi-gaps.py --check
go test ./...
make coverage-baseline
go tool cover -func=/tmp/graviteeam-coverage.out
go vet ./...
go build ./...
go mod verify
go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...
make docs-check
make lint
```

Acceptance coverage is validated by the GitHub Actions test workflow on `main`.
