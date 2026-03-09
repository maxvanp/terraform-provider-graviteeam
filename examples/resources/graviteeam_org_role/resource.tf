resource "graviteeam_org_role" "example" {
  name            = "Domain Manager"
  description     = "Manages a specific domain"
  assignable_type = "DOMAIN"
  permissions     = ["DOMAIN_READ", "DOMAIN_SETTINGS"]
}
