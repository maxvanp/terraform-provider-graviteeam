resource "graviteeam_authorization_engine" "example" {
  domain_id     = graviteeam_domain.example.id
  name          = "OpenFGA authorization engine"
  type          = "openfga"
  configuration = jsonencode({})
}
