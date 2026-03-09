resource "graviteeam_reporter" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "File Audit Reporter"
  type      = "reporter-am-file"
  enabled   = true
  configuration = jsonencode({
    filename = "audit-events.log"
  })
}
