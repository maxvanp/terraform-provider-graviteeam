data "graviteeam_plugins" "identity_providers" {
  category = "identities"
}

data "graviteeam_plugins" "http_identity_provider_schema" {
  category  = "identities"
  plugin_id = "http-am-idp"
  schema    = true
}

data "graviteeam_plugins" "policy_documentation" {
  category      = "policies"
  plugin_id      = "groovy"
  documentation = true
}
