resource "graviteeam_domain" "example" {
  name = "customer-portal"
  oidc {}
  login_settings {}
}

resource "graviteeam_org_user" "example" {
  username         = "domain-admin"
  password         = "SecurePass123!"
  email            = "domain-admin@example.com"
  first_name       = "Domain"
  last_name        = "Admin"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_role" "example" {
  name            = "Domain Support"
  description     = "Support role for a specific domain"
  assignable_type = "DOMAIN"
}

resource "graviteeam_domain_member" "example" {
  domain_id   = graviteeam_domain.example.id
  member_id   = graviteeam_org_user.example.id
  member_type = "USER"
  role_id     = graviteeam_org_role.example.id
}
