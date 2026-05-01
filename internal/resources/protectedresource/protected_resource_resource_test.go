package protectedresource_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccProtectedResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-protected-resource"
  description = "Domain for protected resource acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_protected_resource" "test" {
  domain_id            = graviteeam_domain.test.id
  name                 = "Test MCP Server"
  type                 = "MCP_SERVER"
  description          = "Acceptance test protected resource"
  resource_identifiers = ["https://api.example.com/mcp"]

  feature {
    key         = "list_items"
    type        = "MCP_TOOL"
    description = "List items"
    scopes      = ["openid"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_protected_resource.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_protected_resource.test", "domain_id"),
					resource.TestCheckResourceAttrSet("graviteeam_protected_resource.test", "client_id"),
					resource.TestCheckResourceAttrSet("graviteeam_protected_resource.test", "client_secret"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "name", "Test MCP Server"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "type", "MCP_SERVER"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "description", "Acceptance test protected resource"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "resource_identifiers.#", "1"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "feature.#", "1"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "feature.0.key", "list_items"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "feature.0.type", "MCP_TOOL"),
				),
			},
			{
				ResourceName:      "graviteeam_protected_resource.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_protected_resource.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_protected_resource.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"client_secret", "settings_json"},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-protected-resource"
  description = "Domain for protected resource acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_protected_resource" "test" {
  domain_id            = graviteeam_domain.test.id
  name                 = "Updated MCP Server"
  type                 = "MCP_SERVER"
  description          = "Updated protected resource"
  resource_identifiers = ["https://api.example.com/mcp", "https://api.example.com/mcp/secondary"]

  feature {
    key         = "list_items"
    type        = "MCP_TOOL"
    description = "Updated list items"
    scopes      = ["openid", "profile"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "name", "Updated MCP Server"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "description", "Updated protected resource"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "resource_identifiers.#", "2"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "feature.#", "1"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource.test", "feature.0.description", "Updated list items"),
				),
			},
		},
	})
}
