resource "graviteeam_device_identifier" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "FingerprintJS"
  type      = "fingerprintjs-v3-am-device-identifier"
  configuration = jsonencode({})
}
