resource "graviteeam_group_members" "example" {
  domain_id = graviteeam_domain.example.id
  group_id  = graviteeam_group.example.id
  members   = [graviteeam_user.example.id]
}
