data "graviteeam_application_metadata" "analytics" {
  domain_id      = graviteeam_domain.example.id
  application_id = graviteeam_application.example.id
  kind           = "analytics"
  type           = "GROUP_BY"
  field          = "application"
  interval       = 86400000
}

data "graviteeam_application_metadata" "resources" {
  domain_id      = graviteeam_domain.example.id
  application_id = graviteeam_application.example.id
  kind           = "resources"
  size           = 50
}

data "graviteeam_application_metadata" "resource_policies" {
  domain_id      = graviteeam_domain.example.id
  application_id = graviteeam_application.example.id
  resource_id    = "resource-id"
  kind           = "resource_policies"
}
