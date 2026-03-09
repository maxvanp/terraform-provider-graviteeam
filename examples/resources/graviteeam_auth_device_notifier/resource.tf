resource "graviteeam_auth_device_notifier" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "HTTP Notifier"
  type      = "http-am-authdevice-notifier"
  configuration = jsonencode({
    endpoint    = "https://example.com/ciba/notify"
    headerName  = "Authorization"
    headerValue = "Bearer token"
  })
}
