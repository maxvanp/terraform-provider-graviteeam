# Gravitee AM API Coverage

This document tracks the provider coverage against the bundled Gravitee AM Management API reference.

## Source

| Item | Value |
|------|-------|
| Target Gravitee AM version | `4.11.4` |
| Bundled OpenAPI file | [`docs/openapi.yaml`](openapi.yaml) |
| Bundled OpenAPI version | `4.11.4` |
| OpenAPI path entries | `201` |
| Path entries with at least one write verb | `123` |
| Terraform resources registered | `51` |
| Terraform data sources registered | `13` |

The OpenAPI file is a reference snapshot only. Refresh it from the official Gravitee repository with:

```bash
./scripts/update-openapi.sh
```

Run the coverage audit with:

```bash
./scripts/audit-openapi-coverage.py
```

Current automated audit summary:

| Item | Value |
|------|-------|
| OpenAPI families | `132` |
| Writable families | `77` |
| Read-only families | `55` |
| Uncovered writable families without Terraform resource | `17` |
| Writable families covered only by data source | `5` |
| Uncovered read-only families | `16` |
| Registered resources missing test/doc/example artifact | `0` |
| Registered data sources missing test/doc/example artifact | `0` |

Coverage below is grouped by API family, not by every individual OpenAPI path. Some API paths are action-only or operational endpoints and are not good Terraform resource candidates. Writable families require a Terraform resource to count as covered; a read-only data source is useful, but it does not make the writable API family fully managed.

Current unit coverage baseline:

| Item | Value |
|------|-------|
| `go test ./... -coverprofile=/tmp/graviteeam-coverage.out -covermode=atomic` | `3.6%` total statement coverage |

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
| `domain:applications/type` | `graviteeam_application` |
| `domain:authorization-engines` | `graviteeam_authorization_engine` |
| `domain:auth-device-notifiers` | `graviteeam_auth_device_notifier` |
| `domain:bot-detections` | `graviteeam_bot_detection` |
| `domain:certificates` | `graviteeam_certificate` |
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
| `domain:reporters` | `graviteeam_reporter` |
| `domain:resources` | `graviteeam_service_resource` |
| `domain:roles` | `graviteeam_role` |
| `domain:scopes` | `graviteeam_scope` |
| `domain:themes` | `graviteeam_theme` |
| `domain:users` | `graviteeam_user` |
| `domain:users/cert-credentials` | `graviteeam_user_certificate_credential` |
| `domain:users/lock` | `graviteeam_user` |
| `domain:users/roles` | `graviteeam_user_role` |
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
| `org:users/status` | `graviteeam_org_user` |
| `org:users/tokens` | `graviteeam_org_user_token` |
| `org:users/username` | `graviteeam_org_user` |

## Covered Data Sources

| API family | Terraform data source |
|------------|-----------------------|
| `domain:analytics` | `graviteeam_analytics` |
| `domain:audits` | `graviteeam_audits` |
| `domain:certificates/key` | `graviteeam_domain_metadata` |
| `domain:certificates/keys` | `graviteeam_domain_metadata` |
| `domain:entrypoints` | `graviteeam_entrypoints` |
| `domain:flows` | `graviteeam_flows` |
| `domain:password-policies/activePolicy` | `graviteeam_domain_metadata` |
| `environment:data-planes` | `graviteeam_environment_metadata` |
| `environment:data-sources` | `graviteeam_environment_metadata` |
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
| `platform:plugins/policies/schema` | `graviteeam_plugins` |
| `platform:plugins/reporters` | `graviteeam_plugins` |
| `platform:plugins/reporters/schema` | `graviteeam_plugins` |
| `platform:plugins/resources` | `graviteeam_plugins` |
| `platform:plugins/resources/schema` | `graviteeam_plugins` |
| `domain:users/consents` | `graviteeam_user_consents` |
| `domain:users/credentials` | `graviteeam_user_credentials` |
| `domain:users/devices` | `graviteeam_user_devices` |
| `domain:users/factors` | `graviteeam_user_factors` |
| `domain:users/identities` | `graviteeam_user_identities` |

## Known Gaps

These API families expose write operations in the OpenAPI reference but are not currently modeled as Terraform resources.

### High-Value Resource Candidates

None currently identified in the local 4.11.4 OpenAPI audit.

### Action or Lifecycle Endpoints

These endpoints may be better represented as explicit resources, one-shot actions, or left unmanaged depending on Terraform semantics.

| API family | Notes |
|------------|-------|
| `domain:applications/secrets/_renew` | Application client secret renewal action. |
| `domain:certificates/rotate` | Certificate rotation action. |
| `domain:forms/preview` | Preview action, probably not Terraform-managed. |
| `domain:password-policies/evaluate` | Password policy evaluation action. |
| `domain:protected-resources/secrets/_renew` | Protected resource secret renewal action. |
| `domain:users/bulk` | Bulk user action. |
| `domain:users/consents` | User consent lifecycle; read-only state is exposed by `graviteeam_user_consents`, revocation remains unmanaged. |
| `domain:users/credentials` | User credential lifecycle; read-only state is exposed by `graviteeam_user_credentials`, revocation remains unmanaged. |
| `domain:users/devices` | User device lifecycle; read-only state is exposed by `graviteeam_user_devices`, deletion remains unmanaged. |
| `domain:users/factors` | User factor lifecycle; read-only state is exposed by `graviteeam_user_factors`, revocation remains unmanaged. |
| `domain:users/identities` | User identity lifecycle; read-only state is exposed by `graviteeam_user_identities`, unlink remains unmanaged. |
| `domain:users/resetPassword` | Password reset action. |
| `domain:users/sendRegistrationConfirmation` | Registration confirmation action. |
| `org:users/bulk` | Organization bulk user action. |
| `org:users/resetPassword` | Organization password reset action. |
| `self:newsletter/_subscribe` | Current-user newsletter subscription operation. |
| `self:notifications/acknowledge` | Current-user notification acknowledgement operation. |

### Read-Only and Admin Metadata Not Covered

The provider also does not currently expose several read-only or platform metadata families, including:

| API family | Notes |
|------------|-------|
| `platform:plugins/policies/documentation` | Policy plugin documentation. |
| `platform:roles` | Platform role metadata. |
| `environment:members/permissions` | Environment permissions metadata. |
| `org:audits` | Organization audit logs. |
| `org:environments` | Organization environment listing. |
| `domain:applications/analytics` | Application analytics. |
| `domain:applications/resources` | Application resource listing. |
| `domain:applications/resources/policies` | Application resource policy listing. |
| `domain:users/audits` | User audit logs. |

## Suggested Implementation Order

1. Revisit action-only endpoints and decide case by case whether Terraform should model them.
2. Add read-only data sources for useful runtime and platform metadata families.
3. Reassess local compose plugins when plugin-backed resources are skipped because plugins are unavailable.

## Verification

After changing API coverage, run:

```bash
./scripts/audit-openapi-coverage.py
./scripts/audit-openapi-coverage.py --check-doc
go test ./...
go test ./... -coverprofile=/tmp/graviteeam-coverage.out -covermode=atomic
go tool cover -func=/tmp/graviteeam-coverage.out
go vet ./...
go build ./...
go mod verify
go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...
make docs-check
make lint
```

Acceptance coverage is validated by the GitHub Actions test workflow on `main`.
