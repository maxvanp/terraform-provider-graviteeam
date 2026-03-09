resource "graviteeam_group_roles" "example" {
  domain_id = graviteeam_domain.example.id
  group_id  = graviteeam_group.example.id
  roles     = [graviteeam_role.example.id]
}
