resource "graviteeam_factor" "example" {
  domain_id   = graviteeam_domain.example.id
  name        = "TOTP Authenticator"
  factor_type = "TOTP"
}
