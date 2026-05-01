resource "graviteeam_protected_resource" "example" {
  domain_id            = graviteeam_domain.example.id
  name                 = "MCP Server"
  type                 = "MCP_SERVER"
  description          = "Example protected MCP server"
  resource_identifiers = ["https://api.example.com/mcp"]

  feature {
    key         = "list_items"
    type        = "MCP_TOOL"
    description = "List items"
    scopes      = ["openid"]
  }
}
