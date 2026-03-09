resource "graviteeam_user_role" "example" {
  domain_id = graviteeam_domain.example.id
  user_id   = graviteeam_user.example.id
  roles     = [graviteeam_role.example.id]
}
