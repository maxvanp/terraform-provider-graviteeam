resource "graviteeam_form" "example" {
  domain_id = graviteeam_domain.example.id
  template  = "LOGIN"
  enabled   = true
  content   = "<html><body><h1>Welcome</h1></body></html>"
}
