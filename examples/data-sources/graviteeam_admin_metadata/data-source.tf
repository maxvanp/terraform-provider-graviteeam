data "graviteeam_admin_metadata" "organization_audits" {
  kind = "organization_audits"
  size = 10
}

data "graviteeam_admin_metadata" "organization_environments" {
  kind = "organization_environments"
}

data "graviteeam_admin_metadata" "domain_by_hrid" {
  kind = "domain_by_hrid"
  hrid = graviteeam_domain.example.name
}

data "graviteeam_admin_metadata" "user_audits" {
  kind      = "user_audits"
  domain_id = graviteeam_domain.example.id
  user_id   = graviteeam_user.example.id
  size      = 10
}
