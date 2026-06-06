resource "graviteeam_user" "example" {
  domain_id        = graviteeam_domain.example.id
  username         = "testuser"
  email            = "test@example.com"
  first_name       = "Test"
  last_name        = "User"
  enabled          = true
  pre_registration = true

  reset_password         = "SecurePass123!"
  reset_password_trigger = "rotation-2026-01"
  registration_confirmation_trigger = "registration-email-2026-01"
}
