resource "graviteeam_bot_detection" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "reCAPTCHA v3"
  type      = "google-recaptcha-v3-am-bot-detection"
  configuration = jsonencode({
    siteKey   = "6Le..."
    secretKey = "6Le..."
  })
}
