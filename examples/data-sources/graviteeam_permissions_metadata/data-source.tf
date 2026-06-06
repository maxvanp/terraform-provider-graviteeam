data "graviteeam_permissions_metadata" "environment" {
  kind = "environment_member_permissions"
}

data "graviteeam_permissions_metadata" "domain" {
  kind      = "domain_member_permissions"
  domain_id = graviteeam_domain.example.id
}

data "graviteeam_permissions_metadata" "application" {
  kind           = "application_member_permissions"
  domain_id      = graviteeam_domain.example.id
  application_id = graviteeam_application.example.id
}

data "graviteeam_permissions_metadata" "protected_resource" {
  kind                  = "protected_resource_member_permissions"
  domain_id             = graviteeam_domain.example.id
  protected_resource_id = graviteeam_protected_resource.example.id
}
