# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - TBD

### Added

- Provider configuration with OAuth2 authentication (`api_url`, `client_id`, `client_secret`, `organization_id`, `environment_id`)
- 21 resources:
  - `graviteeam_domain` — Security domain (equivalent Keycloak Realm)
  - `graviteeam_application` — OAuth2/OIDC application with IdP rules, MFA, OAuth settings
  - `graviteeam_identity_provider` — Identity provider (inline, JDBC, HTTP, OAuth2) with mappers and role mapping
  - `graviteeam_factor` — MFA factor (TOTP, EMAIL, SMS)
  - `graviteeam_user` — Domain user with pre-registration support
  - `graviteeam_password_policy` — Password complexity rules
  - `graviteeam_scope` — OAuth2 scope
  - `graviteeam_role` — Domain-level role with OAuth scopes
  - `graviteeam_group` — User group with roles and members
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
  - `graviteeam_alert_notifier` — Alert webhook notifier
- 4 data sources:
  - `graviteeam_analytics` — Read analytics data
  - `graviteeam_audits` — Read audit logs
  - `graviteeam_entrypoints` — Read domain entrypoints
  - `graviteeam_flows` — Read domain flows
- Import support for all resources (`terraform import`)
