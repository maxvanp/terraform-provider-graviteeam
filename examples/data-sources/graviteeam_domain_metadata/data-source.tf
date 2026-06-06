data "graviteeam_domain_metadata" "active_password_policy" {
  domain_id = graviteeam_domain.example.id
  kind      = "active_password_policy"
}

data "graviteeam_domain_metadata" "certificate_keys" {
  domain_id      = graviteeam_domain.example.id
  kind           = "certificate_keys"
  certificate_id = graviteeam_certificate.example.id
}
