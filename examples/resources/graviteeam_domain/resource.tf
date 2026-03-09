resource "graviteeam_domain" "example" {
  name        = "my-domain"
  description = "My Security Domain"
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
