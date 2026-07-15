resource "graviteeam_reporter" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "File Audit Reporter"
  type      = "reporter-am-file"
  enabled   = true
  inherited = false
  configuration = jsonencode({
    filename = "audit-events.log"
  })
}
