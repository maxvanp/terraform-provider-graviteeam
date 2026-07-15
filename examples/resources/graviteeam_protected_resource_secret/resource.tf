resource "graviteeam_domain" "example" {
  name = "customer-portal"
  oidc {}
  login_settings {}
}

resource "graviteeam_protected_resource" "example" {
  domain_id            = graviteeam_domain.example.id
  name                 = "MCP Server"
  type                 = "MCP_SERVER"
  resource_identifiers = ["https://api.example.com/mcp"]

  feature {
    key         = "list_items"
    type        = "MCP_TOOL"
    description = "List items"
    scopes      = ["openid"]
  }
}

resource "graviteeam_protected_resource_secret" "example" {
  domain_id             = graviteeam_domain.example.id
  protected_resource_id = graviteeam_protected_resource.example.id
  name                  = "terraform-managed"
  renew_trigger         = "rotation-2026-01"
}
