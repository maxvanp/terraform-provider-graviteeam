resource "graviteeam_service_resource" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "SMTP Server"
  type      = "smtp-am-resource"
  configuration = jsonencode({
    host           = "smtp.example.com"
    port           = 587
    from           = "noreply@example.com"
    protocol       = "smtp"
    authentication = true
    startTls       = true
  })
}
