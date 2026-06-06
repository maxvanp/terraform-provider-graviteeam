data "graviteeam_user_consents" "example" {
  domain_id = graviteeam_domain.example.id
  user_id   = graviteeam_user.example.id
}
