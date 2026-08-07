# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Align README/CONTRIBUTING with the current Go 1.26.5 toolchain requirement and full resource inventory
- Add `group_mapper` support to `graviteeam_identity_provider`
- Add an OpenAPI coverage audit script and document the current resource/data source gaps
- Add `graviteeam_org_group` for organization-level group management
- Add `graviteeam_domain_flow` for domain-level authentication flow management
- Add `graviteeam_org_reporter` for organization-level reporter management
- Add `graviteeam_org_form` for organization-level form templates
- Add `graviteeam_org_entrypoint` for organization-level entrypoint management
- Add `graviteeam_org_user` for organization-level user management with a sensitive initial password
- Add `graviteeam_org_group_members` for organization-level group membership management
- Add `graviteeam_org_member` for organization-level membership role assignments
- Add `graviteeam_domain_member` for domain membership role assignments
- Add `graviteeam_application_member` for application membership role assignments
- Add `graviteeam_protected_resource_member` for protected resource membership role assignments
- Add `graviteeam_application_secret` for application client secret lifecycle management
- Add `graviteeam_protected_resource_secret` for protected resource client secret lifecycle management
- Add `graviteeam_identity_provider_password_policy` for identity provider password policy assignments
- Add `graviteeam_domain_certificate_settings` for domain fallback certificate settings
- Add `graviteeam_generated_certificate` for generated system certificate lifecycle management
- Add `graviteeam_alert_trigger` for domain alert trigger configuration
- Add `graviteeam_org_user_token` for organization user account token lifecycle management
- Add `graviteeam_user_certificate_credential` for domain user certificate credential lifecycle management
- Add `graviteeam_authorization_engine` for domain authorization engine plugin management
- Add user runtime collection data sources for consents, credentials, devices, factors, and identities
- Add `graviteeam_plugins` for platform plugin catalogs, plugin details, and plugin schemas
- Add platform and environment metadata data sources
- Add `graviteeam_domain_metadata` for active password policy and certificate public key metadata
- Add `graviteeam_permissions_metadata` for environment, domain, application, and protected resource member permission metadata
- Add `graviteeam_admin_metadata` for organization audits, organization environments, domain HRID lookup, and user audits
- Add `graviteeam_application_metadata` for application analytics and UMA resources
- Add `graviteeam_self_metadata` for authenticated user profile and notifications
- Add `graviteeam_trust_domain` for workload identity trust domains backed by a JWKS URL
- Add `graviteeam_applications` for page and cursor application search
- Add `graviteeam_form_preview` and `graviteeam_password_policy_evaluation` read-like POST data sources
- Add policy plugin documentation support to `graviteeam_plugins`
- Add `metadata_json` support to `graviteeam_application`
- Add `settings_json` support to `graviteeam_domain` for advanced domain patch settings

### Changed

- Update `google.golang.org/grpc` to 1.82.1 to remediate GO-2026-6061
- Pin GoReleaser and validate complete unsigned release snapshots in CI
- Updated the local test stack and compatibility target from Gravitee AM 4.6.x to 4.12.1
- Refreshed the bundled Gravitee AM Management API reference from the 4.12.1 upstream tag
- Updated Go module dependencies to their latest compatible versions
- Fixed provider updates for resources that now require plugin `type` or `dataPlaneId` fields with newer Gravitee AM Management API versions
- Use the dedicated i18n dictionary entries endpoint when managing dictionary entries
- Use the dedicated password policy default endpoint when setting a domain default policy
- Add `enabled` management to `graviteeam_user` through the dedicated user status endpoint
- Use the dedicated organization user status endpoint when managing `graviteeam_org_user.enabled`
- Use dedicated username endpoints when updating `graviteeam_user.username` and `graviteeam_org_user.username`
- Add explicit reset password triggers to `graviteeam_user` and `graviteeam_org_user`, and a registration confirmation trigger to `graviteeam_user`
- Use the dedicated application type endpoint when updating `graviteeam_application.type`
- Manage `graviteeam_user.locked` with the dedicated user lock and unlock endpoints
- Add application `kind` support and avoid re-sending incompatible API default settings after application creation
- Preserve server-derived user display names after creation and hydrate typed application settings during import
- Harden CI and release engineering with current pinned actions, full-history secret scanning, Terraform 1.15.8 acceptance tests, and the Registry platform artifact matrix
- Update indirect Go dependencies

## [0.1.0] - TBD

### Added

- Provider configuration with OAuth2 authentication (`api_url`, `client_id`, `client_secret`, `organization_id`, `environment_id`)
- 53 resources:
  - `graviteeam_domain` — Security domain (equivalent Keycloak Realm)
  - `graviteeam_domain_certificate_settings` — Domain fallback certificate settings
  - `graviteeam_generated_certificate` — Generated system certificate
  - `graviteeam_domain_flow` — Domain authentication flow list
  - `graviteeam_domain_member` — Domain membership role assignment
  - `graviteeam_application` — OAuth2/OIDC application with IdP rules, MFA, OAuth settings
  - `graviteeam_application_email` — Application email templates and overrides
  - `graviteeam_application_flow` — Application-specific flows
  - `graviteeam_application_form` — Application-specific forms
  - `graviteeam_application_member` — Application membership role assignment
  - `graviteeam_application_secret` — Application client secret
  - `graviteeam_identity_provider` — Identity provider (inline, JDBC, HTTP, OAuth2) with mappers and role mapping
  - `graviteeam_identity_provider_password_policy` — Identity provider password policy assignment
  - `graviteeam_factor` — MFA factor (TOTP, EMAIL, SMS)
  - `graviteeam_user` — Domain user with pre-registration support
  - `graviteeam_user_certificate_credential` — Domain user certificate credential
  - `graviteeam_password_policy` — Password complexity rules
  - `graviteeam_protected_resource` — Protected MCP server resource with generated OAuth credentials
  - `graviteeam_protected_resource_member` — Protected resource membership role assignment
  - `graviteeam_protected_resource_secret` — Protected resource client secret
  - `graviteeam_scope` — OAuth2 scope
  - `graviteeam_role` — Domain-level role with OAuth scopes
  - `graviteeam_group` — User group with roles and members
  - `graviteeam_group_members` — Group membership management
  - `graviteeam_group_roles` — Group role assignments
  - `graviteeam_theme` — UI branding (colors, CSS, logo)
  - `graviteeam_trust_domain` — Workload identity trust domain backed by a JWKS URL
  - `graviteeam_form` — Custom page template (LOGIN, REGISTRATION, etc.)
  - `graviteeam_email_template` — Custom email template
  - `graviteeam_extension_grant` — Custom grant type (JWT Bearer)
  - `graviteeam_certificate` — Certificate for JWT signing (PKCS12)
  - `graviteeam_reporter` — Audit reporter plugin
  - `graviteeam_service_resource` — Shared resource plugin (SMTP, etc.)
  - `graviteeam_bot_detection` — Bot detection plugin
  - `graviteeam_device_identifier` — Device fingerprinting plugin
  - `graviteeam_auth_device_notifier` — CIBA auth device notifier
  - `graviteeam_i18n_dictionary` — Internationalization dictionary
  - `graviteeam_org_entrypoint` — Organization-level entrypoint
  - `graviteeam_org_form` — Organization-level form template
  - `graviteeam_org_group` — Organization-level group
  - `graviteeam_org_group_members` — Organization-level group membership management
  - `graviteeam_org_identity_provider` — Organization-level identity provider
  - `graviteeam_org_member` — Organization-level membership role assignment
  - `graviteeam_org_reporter` — Organization-level reporter
  - `graviteeam_org_role` — Organization-level role
  - `graviteeam_org_settings` — Organization settings singleton
  - `graviteeam_org_tag` — Organization tag
  - `graviteeam_org_user` — Organization-level user
  - `graviteeam_org_user_token` — Organization user account token
  - `graviteeam_alert_notifier` — Alert webhook notifier
  - `graviteeam_alert_trigger` — Domain alert trigger configuration
  - `graviteeam_user_role` — User role assignments
- 20 data sources:
  - `graviteeam_applications` — Search applications with page or cursor pagination
  - `graviteeam_analytics` — Read analytics data
  - `graviteeam_audits` — Read audit logs
  - `graviteeam_entrypoints` — Read domain entrypoints
  - `graviteeam_flows` — Read domain flows
  - `graviteeam_form_preview` — Render a domain form or email template preview
  - `graviteeam_password_policy_evaluation` — Evaluate a password against a domain password policy
- Import support for all resources (`terraform import`)
