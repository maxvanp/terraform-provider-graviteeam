resource "graviteeam_email_template" "example" {
  domain_id     = graviteeam_domain.example.id
  template      = "REGISTRATION_CONFIRMATION"
  enabled       = true
  from          = "noreply@example.com"
  from_name     = "My App"
  subject       = "Confirm your registration"
  content       = "<html><body>Click to confirm: ${token}</body></html>"
  expires_after = 86400
}
