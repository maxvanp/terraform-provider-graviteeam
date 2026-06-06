resource "graviteeam_org_user" "example" {
  username         = "org-token-user"
  password         = "SecurePass123!"
  email            = "org-token-user@example.com"
  first_name       = "Org"
  last_name        = "Token"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_user_token" "example" {
  user_id = graviteeam_org_user.example.id
  name    = "automation-token"
}
