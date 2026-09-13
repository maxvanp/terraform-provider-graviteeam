# Terraform Provider for Gravitee Access Management

A Terraform provider for managing [Gravitee Access Management](https://www.gravitee.io/platform/access-management) resources. This provider enables you to configure security domains, applications, identity providers, MFA factors, and more through infrastructure as code.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.26.8 (for building from source)
- [Gravitee Access Management](https://www.gravitee.io/platform/access-management) 4.12.6 (the tested compatibility target)

## Compatibility

This provider targets the **Gravitee AM 4.12.6** Management API. Acceptance tests are run against Gravitee AM 4.12.6.

The bundled [`docs/openapi.yaml`](docs/openapi.yaml) file is a local API reference and is not used by the provider runtime or CI for compatibility validation. Refresh it from the official Gravitee source with `./scripts/update-openapi.sh` when you bump the target AM version.

Current API coverage is intentionally partial. See [`docs/api-coverage.md`](docs/api-coverage.md) for the provider coverage matrix and known OpenAPI gaps.

## Maintenance Status

This provider is currently **community-maintained** and released on a **best-effort** basis for Gravitee AM use cases covered by the acceptance suite.

Until `v1.0.0`, compatibility may still evolve as the Gravitee AM API changes. Any intentional breaking change should be called out in [CHANGELOG.md](CHANGELOG.md). If the project stops being maintained, the repository and README should be updated to make that status explicit.

## Installation

Download the signed archives from [GitHub Releases](https://github.com/maxvanp/terraform-provider-graviteeam/releases/tag/v0.1.0). After initial Registry registration, the pinned `required_providers` example below works with `terraform init` directly.

### Install a GitHub release

Before Registry registration, use a filesystem mirror. Download the ZIP for your platform, the `SHA256SUMS` file, and its `.sig` from the release. Verify the signature with the [public signing key](docs/release-signing-key.asc), whose fingerprint is `96BE5305AEB21B9D1877B2E40C698A4920F58A9A`, then verify the ZIP checksum.

Place the ZIP in `/absolute/path/to/mirror/registry.terraform.io/maxvanp/graviteeam/` without renaming it. Save this Terraform CLI configuration as `mirror.tfrc` in your working directory:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/absolute/path/to/mirror"
    include = ["registry.terraform.io/maxvanp/graviteeam"]
  }
  direct {
    exclude = ["registry.terraform.io/maxvanp/graviteeam"]
  }
}
```

Use the version-pinned configuration below, then run `TF_CLI_CONFIG_FILE="$PWD/mirror.tfrc" terraform init`. Commit the resulting `.terraform.lock.hcl`. This mirror configuration applies only to this command and does not change your global CLI settings.

### Local Development (dev_overrides)

Build the provider and configure `~/.terraformrc` to use the local binary:

```bash
git clone https://github.com/maxvanp/terraform-provider-graviteeam.git
cd terraform-provider-graviteeam
make build
```

Add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "maxvanp/graviteeam" = "/path/to/terraform-provider-graviteeam"
  }
  direct {}
}
```

## Usage

```hcl
terraform {
  required_providers {
    graviteeam = {
      source  = "maxvanp/graviteeam"
      version = "0.1.0"
    }
  }
}

provider "graviteeam" {
  api_url       = "http://localhost:8093"
  client_id     = "admin"
  client_secret = "adminadmin"
}

resource "graviteeam_domain" "example" {
  name        = "my-domain"
  description = "Example security domain"
  enabled     = true

  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }

  login_settings {
    register_enabled        = true
    forgot_password_enabled = true
  }
}
```

Until Registry registration is complete, use the local `dev_overrides` workflow above or a Terraform CLI filesystem mirror for the signed GitHub archive.

## Resources and Data Sources

### Resources (53)

| Resource | Description |
|----------|-------------|
| `graviteeam_alert_trigger` | Domain alert trigger configuration |
| `graviteeam_domain` | Security domain (equivalent Keycloak Realm) |
| `graviteeam_domain_certificate_settings` | Domain fallback certificate settings |
| `graviteeam_application` | OAuth2/OIDC application with IdP rules, MFA, OAuth settings, and raw advanced settings JSON |
| `graviteeam_application_email` | Application email templates and overrides |
| `graviteeam_application_flow` | Application-specific flows |
| `graviteeam_application_form` | Application-specific forms |
| `graviteeam_application_member` | Application membership role assignment |
| `graviteeam_application_secret` | Application client secret |
| `graviteeam_identity_provider` | Identity provider (inline, JDBC, HTTP, OAuth2) with mappers |
| `graviteeam_identity_provider_password_policy` | Identity provider password policy assignment |
| `graviteeam_factor` | MFA factor (TOTP, EMAIL, SMS) |
| `graviteeam_user` | Domain user with pre-registration support |
| `graviteeam_user_certificate_credential` | Domain user certificate credential |
| `graviteeam_password_policy` | Password complexity rules |
| `graviteeam_protected_resource` | Protected MCP server resource with generated OAuth credentials |
| `graviteeam_protected_resource_member` | Protected resource membership role assignment |
| `graviteeam_protected_resource_secret` | Protected resource client secret |
| `graviteeam_scope` | OAuth2 scope |
| `graviteeam_role` | Domain-level role with OAuth scopes |
| `graviteeam_group` | User group with roles and members |
| `graviteeam_group_members` | Group membership management |
| `graviteeam_group_roles` | Group role assignments |
| `graviteeam_theme` | UI branding (colors, CSS, logo) |
| `graviteeam_trust_domain` | Workload identity trust domain backed by a JWKS URL |
| `graviteeam_form` | Custom page template (LOGIN, REGISTRATION, etc.) |
| `graviteeam_email_template` | Custom email template |
| `graviteeam_extension_grant` | Custom grant type (JWT Bearer) |
| `graviteeam_certificate` | Certificate for JWT signing (PKCS12) |
| `graviteeam_generated_certificate` | Generated system certificate |
| `graviteeam_reporter` | Audit reporter plugin |
| `graviteeam_service_resource` | Shared resource plugin (SMTP, etc.) |
| `graviteeam_authorization_engine` | Authorization engine plugin |
| `graviteeam_bot_detection` | Bot detection plugin |
| `graviteeam_device_identifier` | Device fingerprinting plugin |
| `graviteeam_domain_flow` | Domain authentication flow list |
| `graviteeam_domain_member` | Domain membership role assignment |
| `graviteeam_auth_device_notifier` | CIBA auth device notifier |
| `graviteeam_i18n_dictionary` | Internationalization dictionary |
| `graviteeam_org_entrypoint` | Organization-level entrypoint |
| `graviteeam_org_form` | Organization-level form template |
| `graviteeam_org_group` | Organization-level group |
| `graviteeam_org_group_members` | Organization-level group membership management |
| `graviteeam_org_identity_provider` | Organization-level identity provider |
| `graviteeam_org_member` | Organization-level membership role assignment |
| `graviteeam_org_reporter` | Organization-level reporter |
| `graviteeam_org_role` | Organization-level role |
| `graviteeam_org_settings` | Organization settings singleton |
| `graviteeam_org_tag` | Organization tag |
| `graviteeam_org_user` | Organization-level user |
| `graviteeam_org_user_token` | Organization user account token |
| `graviteeam_alert_notifier` | Alert webhook notifier |
| `graviteeam_user_role` | User role assignments |

### Data Sources (20)

| Data Source | Description |
|-------------|-------------|
| `graviteeam_admin_metadata` | Read administrative metadata such as organization audits, environments, domain HRID lookups, and user audits |
| `graviteeam_analytics` | Read analytics data (DATE_HISTO, COUNT, GROUP_BY) |
| `graviteeam_application_metadata` | Read application analytics and UMA resources |
| `graviteeam_applications` | Search applications with page or cursor pagination |
| `graviteeam_audits` | Read audit logs for a domain |
| `graviteeam_domain_metadata` | Read domain metadata such as active password policy and certificate public keys |
| `graviteeam_entrypoints` | Read domain entrypoints |
| `graviteeam_environment_metadata` | Read environment metadata |
| `graviteeam_flows` | Read domain flows |
| `graviteeam_form_preview` | Render a domain form or email template preview |
| `graviteeam_password_policy_evaluation` | Evaluate a password against a domain password policy |
| `graviteeam_permissions_metadata` | Read member permission metadata |
| `graviteeam_platform_metadata` | Read platform metadata and configuration schemas |
| `graviteeam_plugins` | Read platform plugin catalogs, plugin details, and plugin schemas |
| `graviteeam_self_metadata` | Read authenticated user profile and notifications |
| `graviteeam_user_consents` | Read OAuth consent approvals for a user |
| `graviteeam_user_credentials` | Read credentials for a user |
| `graviteeam_user_devices` | Read registered devices for a user |
| `graviteeam_user_factors` | Read enrolled MFA factors for a user |
| `graviteeam_user_identities` | Read linked identities for a user |

All resources support `terraform import`. See the [documentation](docs/) for details on each resource.

## Development

### Build

```bash
make build
```

### Install locally

```bash
make install
```

### Run tests

```bash
make test       # Unit tests
make testacc    # Acceptance tests (requires running Gravitee AM)
```

### Lint and format

```bash
make fmt        # Format Go code
make vet        # Run go vet
make lint       # Run golangci-lint
```

### Generate documentation

```bash
make docs
```

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.
