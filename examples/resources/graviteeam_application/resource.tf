resource "graviteeam_application" "example" {
  domain_id   = graviteeam_domain.example.id
  name        = "my-app"
  type        = "WEB"
  description = "My Web Application"

  metadata_json = jsonencode({
    owner = {
      team = "iam"
    }
    tenant = {
      id   = "tenant-a"
      name = "Tenant A"
    }
  })

  oauth_settings {
    redirect_uris                  = ["http://localhost:8080/callback"]
    post_logout_redirect_uris      = ["http://localhost:8080/"]
    grant_types                    = ["authorization_code", "refresh_token"]
    response_types                 = ["code"]
    scopes                         = ["openid", "profile", "email"]
    access_token_validity_seconds  = 3600
    refresh_token_validity_seconds = 86400
    id_token_validity_seconds      = 3600
  }

  mfa_settings {
    enrollment = "OPTIONAL"
    challenge  = "REQUIRED"
  }

  settings_json = jsonencode({
    advanced = {
      skipConsent = true
    }
    oauth = {
      forcePKCE               = true
      tokenEndpointAuthMethod = "client_secret_post"
      tokenCustomClaims = [
        {
          claimName  = "tenant"
          claimValue = "{#context.attributes['tenant']}"
          tokenType  = "ACCESS_TOKEN"
        }
      ]
      tokenExchangeOAuthSettings = {
        inherited     = false
        scopeHandling = "DOWNSCOPING"
      }
    }
  })
}
