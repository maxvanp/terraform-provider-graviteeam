resource "graviteeam_application_flow" "example" {
  domain_id      = graviteeam_domain.example.id
  application_id = graviteeam_application.example.id
  flows          = jsonencode([])
}
