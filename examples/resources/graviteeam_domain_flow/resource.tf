resource "graviteeam_domain_flow" "example" {
  domain_id = graviteeam_domain.example.id
  flows     = jsonencode([])
}
