resource "graviteeam_application" "example" {
  domain_id   = graviteeam_domain.example.id
  name        = "my-app"
  type        = "WEB"
  description = "My Web Application"

  oauth_settings {
    redirect_uris             = ["http://localhost:8080/callback"]
    post_logout_redirect_uris = ["http://localhost:8080/"]
    grant_types               = ["authorization_code", "refresh_token"]
    response_types            = ["code"]
    scopes                    = ["openid", "profile", "email"]
  }

  mfa_settings {
    enrollment = "OPTIONAL"
    challenge  = "REQUIRED"
  }
}
