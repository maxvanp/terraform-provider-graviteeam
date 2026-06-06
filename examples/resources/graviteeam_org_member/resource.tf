resource "graviteeam_org_user" "example" {
  username         = "platform-user"
  password         = "SecurePass123!"
  email            = "platform-user@example.com"
  first_name       = "Platform"
  last_name        = "User"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_role" "example" {
  name            = "Platform Support"
  description     = "Support role for organization-level administration"
  assignable_type = "ORGANIZATION"
}

resource "graviteeam_org_member" "example" {
  member_id   = graviteeam_org_user.example.id
  member_type = "USER"
  role_id     = graviteeam_org_role.example.id
}
