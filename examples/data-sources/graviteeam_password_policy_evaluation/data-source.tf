resource "graviteeam_domain" "example" {
  name        = "password-policy-evaluation-domain"
  description = "Domain for password policy evaluation example"

  oidc {}
  login_settings {}
}

resource "graviteeam_password_policy" "example" {
  domain_id  = graviteeam_domain.example.id
  name       = "Evaluation policy"
  min_length = 8
}

data "graviteeam_password_policy_evaluation" "example" {
  domain_id = graviteeam_domain.example.id
  policy_id = graviteeam_password_policy.example.id
  password  = "SecurePass123!"
}
