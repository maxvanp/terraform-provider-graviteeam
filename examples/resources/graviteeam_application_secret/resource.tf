resource "graviteeam_domain" "example" {
  name = "customer-portal"
  oidc {}
  login_settings {}
}

resource "graviteeam_application" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "customer-web"
  type      = "WEB"

  oauth_settings {
    redirect_uris  = ["https://example.com/callback"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
    scopes         = ["openid"]
  }
}

resource "graviteeam_application_secret" "example" {
  domain_id      = graviteeam_domain.example.id
  application_id = graviteeam_application.example.id
  name           = "terraform-managed"
  renew_trigger  = "rotation-2026-01"
}
