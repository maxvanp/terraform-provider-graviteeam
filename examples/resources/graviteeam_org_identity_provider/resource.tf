resource "graviteeam_org_identity_provider" "example" {
  name = "Org Inline IDP"
  type = "inline-am-idp"
  configuration = jsonencode({
    users = []
  })
}
