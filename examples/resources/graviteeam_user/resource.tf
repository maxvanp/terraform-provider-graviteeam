resource "graviteeam_user" "example" {
  domain_id        = graviteeam_domain.example.id
  username         = "testuser"
  email            = "test@example.com"
  first_name       = "Test"
  last_name        = "User"
  pre_registration = true
}
