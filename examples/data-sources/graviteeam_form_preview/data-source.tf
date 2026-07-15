resource "graviteeam_domain" "example" {
  name        = "form-preview-domain"
  description = "Domain for form preview example"

  oidc {}
  login_settings {}
}

data "graviteeam_form_preview" "example" {
  domain_id = graviteeam_domain.example.id
  template  = "LOGIN"
  type      = "FORM"
  content   = "<html><body><h1>Hello</h1></body></html>"
}
