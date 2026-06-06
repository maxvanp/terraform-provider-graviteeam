data "graviteeam_plugins" "identity_providers" {
  category = "identities"
}

data "graviteeam_plugins" "http_identity_provider_schema" {
  category  = "identities"
  plugin_id = "http-am-idp"
  schema    = true
}
