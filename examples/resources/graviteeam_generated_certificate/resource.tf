resource "graviteeam_domain" "example" {
  name        = "generated-certificate-domain"
  description = "Domain for generated certificate example"

  oidc {}
  login_settings {}
}

resource "graviteeam_generated_certificate" "example" {
  domain_id        = graviteeam_domain.example.id
  rotation_trigger = "rotation-2026-01"
}
