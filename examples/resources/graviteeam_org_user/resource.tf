resource "graviteeam_org_user" "example" {
  username         = "platform-user"
  password         = "SecurePass123!"
  email            = "platform-user@example.com"
  first_name       = "Platform"
  last_name        = "User"
  enabled          = true
  pre_registration = true

  reset_password         = "NewSecurePass123!"
  reset_password_trigger = "rotation-2026-01"
}
