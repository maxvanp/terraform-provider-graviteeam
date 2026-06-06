resource "graviteeam_org_form" "example" {
  template = "LOGIN"
  enabled  = true
  content  = "<html><body><h1>Organization Login</h1></body></html>"
}
