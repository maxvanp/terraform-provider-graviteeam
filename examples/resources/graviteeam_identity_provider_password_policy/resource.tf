resource "graviteeam_domain" "example" {
  name        = "idp-password-policy-domain"
  description = "Example domain"

  oidc {}
  login_settings {}
}

resource "graviteeam_password_policy" "example" {
  domain_id  = graviteeam_domain.example.id
  name       = "Example password policy"
  min_length = 12
}

resource "graviteeam_identity_provider_password_policy" "example" {
  domain_id            = graviteeam_domain.example.id
  identity_provider_id = graviteeam_domain.example.default_idp_id
  password_policy_id   = graviteeam_password_policy.example.id
}
