resource "graviteeam_alert_notifier" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "Slack Webhook"
  type      = "webhook-notifier"
  enabled   = true
  configuration = jsonencode({
    url    = "https://hooks.slack.example.com/xxx"
    method = "POST"
    body   = "{\"text\": \"Alert: {{alert.name}}\"}"
  })
}
