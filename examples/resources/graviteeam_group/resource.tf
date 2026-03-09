resource "graviteeam_group" "example" {
  domain_id   = graviteeam_domain.example.id
  name        = "Administrators"
  description = "Admin group"
  roles       = [graviteeam_role.example.id]
  members     = [graviteeam_user.example.id]
}
