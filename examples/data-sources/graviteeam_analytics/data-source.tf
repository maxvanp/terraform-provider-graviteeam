data "graviteeam_analytics" "example" {
  domain_id = graviteeam_domain.example.id
  type      = "COUNT"
  field     = "application"
}
