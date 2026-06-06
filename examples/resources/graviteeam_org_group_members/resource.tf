resource "graviteeam_org_user" "example" {
  username         = "platform-user"
  password         = "SecurePass123!"
  email            = "platform-user@example.com"
  first_name       = "Platform"
  last_name        = "User"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_group" "example" {
  name        = "platform-admins"
  description = "Platform administrators"
}

resource "graviteeam_org_group_members" "example" {
  group_id = graviteeam_org_group.example.id
  members  = [graviteeam_org_user.example.id]
}
