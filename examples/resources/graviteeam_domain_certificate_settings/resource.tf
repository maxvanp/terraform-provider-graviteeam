resource "graviteeam_domain" "example" {
  name        = "certificate-settings-domain"
  description = "Example domain"

  oidc {}
  login_settings {}
}

resource "graviteeam_certificate" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "Fallback certificate"
  type      = "pkcs12-am-certificate"

  configuration = jsonencode({
    content   = jsonencode({ content = "base64-pkcs12-content", name = "certificate.p12" })
    storepass = "changeit"
    alias     = "mykey"
    keypass   = "changeit"
    algorithm = "RS256"
  })
}

resource "graviteeam_domain_certificate_settings" "example" {
  domain_id               = graviteeam_domain.example.id
  fallback_certificate_id = graviteeam_certificate.example.id
}
