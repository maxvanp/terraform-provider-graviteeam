resource "graviteeam_org_entrypoint" "example" {
  name        = "Customer Login"
  description = "Customer-facing login entrypoint"
  url         = "https://login.example.com"
  tags        = [graviteeam_org_tag.example.id]
}
