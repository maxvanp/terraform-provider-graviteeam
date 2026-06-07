resource "graviteeam_user" "example" {
  domain_id            = graviteeam_domain.example.id
  username             = "testuser"
  email                = "test@example.com"
  first_name           = "Test"
  last_name            = "User"
  display_name         = "Test User"
  enabled              = true
  force_reset_password = false
  pre_registration     = true

  reset_password                    = "SecurePass123!"
  reset_password_trigger            = "rotation-2026-01"
  registration_confirmation_trigger = "registration-email-2026-01"
}
