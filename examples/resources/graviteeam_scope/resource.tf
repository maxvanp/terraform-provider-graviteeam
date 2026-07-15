resource "graviteeam_scope" "example" {
  domain_id     = graviteeam_domain.example.id
  key           = "roles"
  name          = "User Roles"
  description   = "Access to user roles"
  discovery     = true
  icon_uri      = "https://example.com/icons/roles.svg"
  parameterized = false
}
