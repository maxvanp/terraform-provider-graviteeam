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

resource "graviteeam_org_user" "example" {
  username         = "protected-resource-admin"
  password         = "SecurePass123!"
  email            = "protected-resource-admin@example.com"
  first_name       = "Protected"
  last_name        = "Admin"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_role" "example" {
  name            = "Protected Resource Support"
  description     = "Support role for a specific protected resource"
  assignable_type = "PROTECTED_RESOURCE"
}

resource "graviteeam_protected_resource_member" "example" {
  domain_id             = graviteeam_domain.example.id
  protected_resource_id = graviteeam_protected_resource.example.id
  member_id             = graviteeam_org_user.example.id
  member_type           = "USER"
  role_id               = graviteeam_org_role.example.id
}
