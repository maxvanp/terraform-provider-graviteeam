data "graviteeam_platform_metadata" "license" {
  kind = "license"
}

data "graviteeam_platform_metadata" "flow_schema" {
  kind = "flow_schema"
}

data "graviteeam_platform_metadata" "role" {
  kind    = "role"
  role_id = "role-id"
}
