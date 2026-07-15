resource "graviteeam_trust_domain" "example" {
  domain_id                = graviteeam_domain.example.id
  name                     = "workloads.example.com"
  description              = "External workload identities"
  bundle_source            = "JWKS_URL"
  jwks_url                 = "https://www.googleapis.com/oauth2/v3/certs"
  refresh_interval_seconds = 300
  allowed_algorithms       = ["RS256"]
}
