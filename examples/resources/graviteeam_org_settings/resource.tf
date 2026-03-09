resource "graviteeam_org_settings" "example" {
  identities = [graviteeam_org_identity_provider.example.id]
}
