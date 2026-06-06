package protectedresourcesecret_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccProtectedResourceSecretResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-protected-resource-secret"
  oidc {}
  login_settings {}
}

resource "graviteeam_protected_resource" "test" {
  domain_id            = graviteeam_domain.test.id
  name                 = "Test Protected Resource Secret"
  type                 = "MCP_SERVER"
  resource_identifiers = ["https://api.example.com/secret-test/mcp"]

  feature {
    key         = "list_items"
    type        = "MCP_TOOL"
    description = "List items"
    scopes      = ["openid"]
  }
}

resource "graviteeam_protected_resource_secret" "test" {
  domain_id             = graviteeam_domain.test.id
  protected_resource_id = graviteeam_protected_resource.test.id
  name                  = "test-acc-protected-resource-secret"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_protected_resource_secret.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_protected_resource_secret.test", "secret"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource_secret.test", "name", "test-acc-protected-resource-secret"),
				),
			},
			{
				ResourceName:      "graviteeam_protected_resource_secret.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_protected_resource_secret.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_protected_resource_secret.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["protected_resource_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{
					"secret",
				},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-protected-resource-secret"
  oidc {}
  login_settings {}
}

resource "graviteeam_protected_resource" "test" {
  domain_id            = graviteeam_domain.test.id
  name                 = "Test Protected Resource Secret"
  type                 = "MCP_SERVER"
  resource_identifiers = ["https://api.example.com/secret-test/mcp"]

  feature {
    key         = "list_items"
    type        = "MCP_TOOL"
    description = "List items"
    scopes      = ["openid"]
  }
}

resource "graviteeam_protected_resource_secret" "test" {
  domain_id             = graviteeam_domain.test.id
  protected_resource_id = graviteeam_protected_resource.test.id
  name                  = "test-acc-protected-resource-secret"
  renew_trigger         = "rotation-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_protected_resource_secret.test", "secret"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource_secret.test", "renew_trigger", "rotation-1"),
				),
			},
		},
	})
}
