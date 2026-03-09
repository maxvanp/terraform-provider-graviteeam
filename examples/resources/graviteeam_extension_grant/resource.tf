resource "graviteeam_extension_grant" "example" {
  domain_id     = graviteeam_domain.example.id
  name          = "JWT Bearer Grant"
  type          = "jwtbearer-am-extension-grant"
  grant_type    = "urn:ietf:params:oauth:grant-type:jwt-bearer"
  configuration = jsonencode({
    publicKey = "ssh-rsa AAAA..."
  })
  create_user = false
  user_exists = true
}
