# Example configuration for the Gravitee AM Terraform Provider
# Showcases a representative sample of the 21 available resources.

terraform {
  required_providers {
    graviteeam = {
      source = "maxvanp/graviteeam"
    }
  }
}

provider "graviteeam" {
  api_url       = "http://localhost:8093"
  client_id     = "admin"
  client_secret = "adminadmin"
}

# --- Domain ---

resource "graviteeam_domain" "lab" {
  name        = "lab-domain"
  description = "Lab Domain for testing"
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

# --- Password Policy ---

resource "graviteeam_password_policy" "strong" {
  domain_id                             = graviteeam_domain.lab.id
  name                                  = "Strong Password Policy"
  min_length                            = 12
  include_numbers                       = true
  include_special_characters            = true
  letters_in_mixed_case                 = true
  exclude_passwords_in_dictionary       = true
  exclude_user_profile_info_in_password = true
  default_policy                        = true
}

# --- Role ---

resource "graviteeam_role" "admin" {
  domain_id       = graviteeam_domain.lab.id
  name            = "ADMIN"
  description     = "Administrator role"
  assignable_type = "DOMAIN"
  oauth_scopes    = ["openid", "profile", "email"]
}

# --- Group ---

resource "graviteeam_group" "admins" {
  domain_id   = graviteeam_domain.lab.id
  name        = "Administrators"
  description = "Mapped administrator group"
  roles       = [graviteeam_role.admin.id]
}

# --- Identity Provider (Inline with group_mapper and role_mapper) ---

resource "graviteeam_identity_provider" "inline" {
  domain_id = graviteeam_domain.lab.id
  name      = "Corporate Directory"
  type      = "inline-am-idp"
  configuration = jsonencode({
    users = [
      {
        firstname = "John"
        lastname  = "Doe"
        username  = "jdoe@example.com"
        email     = "jdoe@example.com"
        password  = "password123" # gitleaks:allow - synthetic example credential
      }
    ]
  })
  mappers = {
    username  = "username"
    email     = "email"
    firstName = "firstname"
    lastName  = "lastname"
  }
  password_policy_id = graviteeam_password_policy.strong.id
  group_mapper = {
    "{#profile['groups'] != null && #profile['groups'].contains('external-admins')}" = [
      graviteeam_group.admins.id
    ]
  }
  role_mapper = {
    "{#profile['username'] != null}" = [graviteeam_role.admin.id]
  }
}

# --- MFA Factors ---

resource "graviteeam_factor" "totp" {
  domain_id   = graviteeam_domain.lab.id
  name        = "TOTP Authenticator"
  factor_type = "TOTP"
}

resource "graviteeam_factor" "email" {
  domain_id   = graviteeam_domain.lab.id
  name        = "Email OTP"
  factor_type = "EMAIL"
}

# --- Application (with identity_provider_rule + mfa_settings) ---

resource "graviteeam_application" "web_app" {
  domain_id   = graviteeam_domain.lab.id
  name        = "example-app"
  type        = "WEB"
  description = "Example Web Application"

  identity_provider_rule {
    identity       = graviteeam_identity_provider.inline.id
    selection_rule = "{true}"
    priority       = 0
  }

  factors = [graviteeam_factor.totp.id, graviteeam_factor.email.id]

  oauth_settings {
    redirect_uris             = ["http://localhost:5000/auth"]
    post_logout_redirect_uris = ["http://localhost:5000/"]
    grant_types               = ["authorization_code", "refresh_token"]
    response_types            = ["code"]
    scopes                    = ["openid", "profile", "email"]
  }

  mfa_settings {
    enrollment = "OPTIONAL"
    challenge  = "REQUIRED"
  }
}

# --- User ---

resource "graviteeam_user" "test" {
  domain_id        = graviteeam_domain.lab.id
  username         = "testuser"
  email            = "test@example.com"
  first_name       = "Test"
  last_name        = "User"
  pre_registration = true
}

# --- Theme ---

resource "graviteeam_theme" "brand" {
  domain_id                  = graviteeam_domain.lab.id
  primary_button_color_hex   = "#1a73e8"
  secondary_button_color_hex = "#e8f0fe"
  primary_text_color_hex     = "#202124"
  secondary_text_color_hex   = "#5f6368"
}

# --- Service Resource (SMTP) ---

resource "graviteeam_service_resource" "smtp" {
  domain_id = graviteeam_domain.lab.id
  name      = "Mailpit SMTP"
  type      = "smtp-am-resource"
  configuration = jsonencode({
    host           = "mailpit"
    port           = 1025
    from           = "noreply@gravitee-lab.local"
    protocol       = "smtp"
    authentication = false
    startTls       = false
  })
}

# --- Alert Notifier (Webhook) ---

resource "graviteeam_alert_notifier" "webhook" {
  domain_id = graviteeam_domain.lab.id
  name      = "Slack Webhook"
  type      = "webhook-notifier"
  enabled   = true
  configuration = jsonencode({
    url    = "https://hooks.slack.example.com/services/xxx"
    method = "POST"
    body   = "{\"text\": \"Alert: {{alert.name}}\"}"
  })
}

# --- Reporter (File) ---

resource "graviteeam_reporter" "file" {
  domain_id = graviteeam_domain.lab.id
  name      = "File Audit Reporter"
  type      = "reporter-am-file"
  enabled   = true
  configuration = jsonencode({
    filename = "audit-events.log"
  })
}

# --- I18n Dictionary ---

resource "graviteeam_i18n_dictionary" "fr" {
  domain_id = graviteeam_domain.lab.id
  name      = "French"
  locale    = "fr"
  entries = {
    "login.title"    = "Connexion"
    "login.username" = "Identifiant"
    "login.password" = "Mot de passe"
  }
}

# --- Outputs ---

output "domain_id" {
  value = graviteeam_domain.lab.id
}

output "app_client_id" {
  value = graviteeam_application.web_app.client_id
}

output "app_client_secret" {
  value     = graviteeam_application.web_app.client_secret
  sensitive = true
}
