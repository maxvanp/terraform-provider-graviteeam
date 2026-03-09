data "graviteeam_audits" "example" {
  domain_id = graviteeam_domain.example.id
  size      = 20
}
