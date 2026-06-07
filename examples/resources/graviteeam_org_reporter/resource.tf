resource "graviteeam_org_reporter" "example" {
  name      = "Organization File Reporter"
  type      = "reporter-am-file"
  enabled   = true
  inherited = false
  configuration = jsonencode({
    filename = "org-audit.log"
  })
}
