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
| Terraform resources registered | `35` |
| Terraform data sources registered | `4` |

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
| Uncovered writable families without Terraform resource | `42` |
| Writable families covered only by data source | `0` |
| Uncovered read-only families | `52` |
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
| `domain:applications` | `graviteeam_application` |
| `domain:applications/emails` | `graviteeam_application_email` |
| `domain:applications/flows` | `graviteeam_application_flow` |
| `domain:applications/forms` | `graviteeam_application_form` |
| `domain:auth-device-notifiers` | `graviteeam_auth_device_notifier` |
| `domain:bot-detections` | `graviteeam_bot_detection` |
| `domain:certificates` | `graviteeam_certificate` |
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
| `domain:identities` | `graviteeam_identity_provider` |
| `domain:password-policies` | `graviteeam_password_policy` |
| `domain:protected-resources` | `graviteeam_protected_resource` |
| `domain:reporters` | `graviteeam_reporter` |
| `domain:resources` | `graviteeam_service_resource` |
| `domain:roles` | `graviteeam_role` |
| `domain:scopes` | `graviteeam_scope` |
| `domain:themes` | `graviteeam_theme` |
| `domain:users` | `graviteeam_user` |
| `domain:users/roles` | `graviteeam_user_role` |
| `environment:domains` | `graviteeam_domain` |
| `org:groups` | `graviteeam_org_group` |
| `org:identities` | `graviteeam_org_identity_provider` |
| `org:reporters` | `graviteeam_org_reporter` |
| `org:roles` | `graviteeam_org_role` |
| `org:settings` | `graviteeam_org_settings` |
| `org:tags` | `graviteeam_org_tag` |

## Covered Data Sources

| API family | Terraform data source |
|------------|-----------------------|
| `domain:analytics` | `graviteeam_analytics` |
| `domain:audits` | `graviteeam_audits` |
| `domain:entrypoints` | `graviteeam_entrypoints` |
| `domain:flows` | `graviteeam_flows` |

## Known Gaps

These API families expose write operations in the OpenAPI reference but are not currently modeled as Terraform resources.

### High-Value Resource Candidates

| API family | Notes |
|------------|-------|
| `domain:protected-resources/members` | Access management for protected resources. |
| `domain:protected-resources/secrets` | Secret lifecycle for protected resources. |
| `domain:authorization-engines` | Full CRUD family for authorization engines. |
| `domain:members` | Domain membership management. |
| `domain:applications/members` | Application membership management. |
| `domain:applications/secrets` | Application secret lifecycle. |
| `domain:alerts/triggers` | Alert trigger configuration. |
| `domain:certificate-settings` | Domain certificate settings. |
| `org:users` | Organization-level user management. |
| `org:groups/members` | Organization group membership management. |
| `org:members` | Organization membership management. |
| `org:forms` | Organization-level form management. |
| `org:entrypoints` | Organization-level entrypoint management. |
| `org:users/tokens` | Organization user token lifecycle. |

### Action or Lifecycle Endpoints

These endpoints may be better represented as explicit resources, one-shot actions, or left unmanaged depending on Terraform semantics.

| API family | Notes |
|------------|-------|
| `domain:applications/type` | Application type change endpoint. |
| `domain:certificates/rotate` | Certificate rotation action. |
| `domain:forms/preview` | Preview action, probably not Terraform-managed. |
| `domain:users/bulk` | Bulk user action. |
| `domain:users/cert-credentials` | User certificate credentials. |
| `domain:users/consents` | User consent lifecycle. |
| `domain:users/credentials` | User credential lifecycle. |
| `domain:users/devices` | User device lifecycle. |
| `domain:users/factors` | User factor lifecycle. |
| `domain:users/identities` | User identity lifecycle. |
| `domain:users/lock` | User lock action. |
| `domain:users/resetPassword` | Password reset action. |
| `domain:users/sendRegistrationConfirmation` | Registration confirmation action. |
| `domain:users/status` | User status action. |
| `domain:users/unlock` | User unlock action. |
| `domain:users/username` | Username update action. |
| `org:users/bulk` | Organization bulk user action. |
| `org:users/resetPassword` | Organization password reset action. |
| `org:users/status` | Organization user status action. |
| `org:users/username` | Organization username update action. |
| `self:newsletter` | Current-user newsletter operation. |
| `self:notifications` | Current-user notification operation. |

### Read-Only and Admin Metadata Not Covered

The provider also does not currently expose several read-only or platform metadata families, including:

| API family | Notes |
|------------|-------|
| `platform:configuration` | Platform configuration metadata. |
| `platform:installation` | Installation metadata. |
| `platform:license` | License metadata. |
| `platform:plugins/*` | Plugin catalogs and schemas. |
| `platform:roles` | Platform role metadata. |
| `environment:data-planes` | Environment data-plane listing. |
| `environment:data-sources` | Environment data-source listing. |
| `environment:members/permissions` | Environment permissions metadata. |
| `org:audits` | Organization audit logs. |
| `org:environments` | Organization environment listing. |
| `domain:applications/analytics` | Application analytics. |
| `domain:applications/resources` | Application resource listing. |
| `domain:applications/resources/policies` | Application resource policy listing. |
| `domain:users/audits` | User audit logs. |
| `domain:users/credentials` | User credential listing. |
| `domain:users/devices` | User device listing. |
| `domain:users/factors` | User factor listing. |
| `domain:users/identities` | User identity listing. |

## Suggested Implementation Order

1. Add protected resource members and secrets.
2. Add authorization engines.
3. Add membership resources for domains, applications, organizations, and organization groups.
4. Add organization-level users, forms, and entrypoints.
5. Revisit action-only endpoints and decide case by case whether Terraform should model them.

## Verification

After changing API coverage, run:

```bash
./scripts/audit-openapi-coverage.py
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
