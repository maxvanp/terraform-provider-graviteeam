# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Align README/CONTRIBUTING with the current Go 1.26.4 toolchain requirement and full resource inventory

### Changed

- Updated the local test stack and compatibility target from Gravitee AM 4.6.x to 4.11.4
- Refreshed the bundled Gravitee AM Management API reference from the 4.11.4 upstream tag
- Updated Go module dependencies to their latest compatible versions
- Fixed provider updates for resources that now require plugin `type` or `dataPlaneId` fields with newer Gravitee AM Management API versions

## [0.1.0] - TBD

### Added

- Provider configuration with OAuth2 authentication (`api_url`, `client_id`, `client_secret`, `organization_id`, `environment_id`)
- 31 resources:
  - `graviteeam_domain` — Security domain (equivalent Keycloak Realm)
  - `graviteeam_application` — OAuth2/OIDC application with IdP rules, MFA, OAuth settings
  - `graviteeam_application_email` — Application email templates and overrides
  - `graviteeam_application_flow` — Application-specific flows
  - `graviteeam_application_form` — Application-specific forms
  - `graviteeam_identity_provider` — Identity provider (inline, JDBC, HTTP, OAuth2) with mappers and role mapping
  - `graviteeam_factor` — MFA factor (TOTP, EMAIL, SMS)
  - `graviteeam_user` — Domain user with pre-registration support
  - `graviteeam_password_policy` — Password complexity rules
  - `graviteeam_scope` — OAuth2 scope
  - `graviteeam_role` — Domain-level role with OAuth scopes
  - `graviteeam_group` — User group with roles and members
  - `graviteeam_group_members` — Group membership management
  - `graviteeam_group_roles` — Group role assignments
  - `graviteeam_theme` — UI branding (colors, CSS, logo)
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
  - `graviteeam_org_identity_provider` — Organization-level identity provider
  - `graviteeam_org_role` — Organization-level role
  - `graviteeam_org_settings` — Organization settings singleton
  - `graviteeam_org_tag` — Organization tag
  - `graviteeam_alert_notifier` — Alert webhook notifier
  - `graviteeam_user_role` — User role assignments
- 4 data sources:
  - `graviteeam_analytics` — Read analytics data
  - `graviteeam_audits` — Read audit logs
  - `graviteeam_entrypoints` — Read domain entrypoints
  - `graviteeam_flows` — Read domain flows
- Import support for all resources (`terraform import`)
