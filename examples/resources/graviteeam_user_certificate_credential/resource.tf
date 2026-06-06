resource "graviteeam_domain" "example" {
  name        = "certificate-credential-domain"
  description = "Domain for certificate credential example"

  oidc {}
  login_settings {}
}

resource "graviteeam_user" "example" {
  domain_id        = graviteeam_domain.example.id
  username         = "certificate-user"
  email            = "certificate-user@example.com"
  first_name       = "Certificate"
  last_name        = "User"
  pre_registration = true
}

resource "graviteeam_user_certificate_credential" "example" {
  domain_id = graviteeam_domain.example.id
  user_id   = graviteeam_user.example.id

  certificate_pem = <<EOT
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
EOT
}
