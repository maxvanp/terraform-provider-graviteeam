resource "graviteeam_password_policy" "example" {
  domain_id                  = graviteeam_domain.example.id
  name                       = "Strong Policy"
  min_length                 = 12
  include_numbers            = true
  include_special_characters = true
  letters_in_mixed_case      = true
  default_policy             = true
}
