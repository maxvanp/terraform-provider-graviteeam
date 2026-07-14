data "graviteeam_applications" "example" {
  domain_id         = graviteeam_domain.example.id
  query             = "customer"
  application_types = ["WEB", "SERVICE"]
  status            = "enabled"
  limit             = 25
  sort              = "updatedAt"
  direction         = "DESC"
}
