resource "graviteeam_certificate" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "JWT Signing Certificate"
  type      = "pkcs12-am-certificate"
  configuration = jsonencode({
    content   = jsonencode({ content = filebase64("cert.p12"), name = "cert.p12" })
    storepass = "changeit"
    alias     = "mykey"
    keypass   = "changeit"
    algorithm = "RS256"
  })
}
