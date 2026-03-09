resource "graviteeam_role" "example" {
  domain_id    = graviteeam_domain.example.id
  name         = "ADMIN"
  description  = "Administrator role"
  oauth_scopes = ["openid", "profile", "email"]
}
