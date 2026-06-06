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
    redirect_uris  = ["http://localhost:8080/callback"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
    scopes         = ["openid"]
  }
}

resource "graviteeam_org_user" "example" {
  username         = "application-admin"
  password         = "SecurePass123!"
  email            = "application-admin@example.com"
  first_name       = "Application"
  last_name        = "Admin"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_role" "example" {
  name            = "Application Support"
  description     = "Support role for a specific application"
  assignable_type = "APPLICATION"
}

resource "graviteeam_application_member" "example" {
  domain_id      = graviteeam_domain.example.id
  application_id = graviteeam_application.example.id
  member_id      = graviteeam_org_user.example.id
  member_type    = "USER"
  role_id        = graviteeam_org_role.example.id
}
