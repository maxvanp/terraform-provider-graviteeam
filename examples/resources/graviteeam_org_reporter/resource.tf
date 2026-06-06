resource "graviteeam_org_reporter" "example" {
  name    = "Organization File Reporter"
  type    = "reporter-am-file"
  enabled = true
  configuration = jsonencode({
    filename = "org-audit.log"
  })
}
