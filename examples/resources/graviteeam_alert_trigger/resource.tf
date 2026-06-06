resource "graviteeam_domain" "example" {
  name        = "alert-trigger-domain"
  description = "Example domain"

  oidc {}
  login_settings {}
}

resource "graviteeam_alert_trigger" "example" {
  domain_id = graviteeam_domain.example.id
  type      = "TOO_MANY_LOGIN_FAILURES"
  enabled   = true
}
