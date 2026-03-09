resource "graviteeam_application_form" "example" {
  domain_id      = graviteeam_domain.example.id
  application_id = graviteeam_application.example.id
  template       = "LOGIN"
  enabled        = true
  content        = "<html><body><h1>Login</h1></body></html>"
}
