package protectedresourcemember_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccProtectedResourceMemberResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-protected-resource-member"
  oidc {}
  login_settings {}
}

resource "graviteeam_protected_resource" "test" {
  domain_id            = graviteeam_domain.test.id
  name                 = "Test Protected Resource Member"
  type                 = "MCP_SERVER"
  resource_identifiers = ["https://api.example.com/member-test/mcp"]

  feature {
    key         = "list_items"
    type        = "MCP_TOOL"
    description = "List items"
    scopes      = ["openid"]
  }
}

resource "graviteeam_org_user" "test" {
  username         = "test-acc-protected-resource-member"
  password         = "SecurePass123!"
  email            = "test-acc-protected-resource-member@example.com"
  first_name       = "Protected"
  last_name        = "Member"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_role" "test" {
  name            = "test-acc-protected-resource-member-role"
  description     = "Acceptance test protected resource member role"
  assignable_type = "PROTECTED_RESOURCE"
}

resource "graviteeam_protected_resource_member" "test" {
  domain_id             = graviteeam_domain.test.id
  protected_resource_id = graviteeam_protected_resource.test.id
  member_id             = graviteeam_org_user.test.id
  member_type           = "USER"
  role_id               = graviteeam_org_role.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_protected_resource_member.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_protected_resource_member.test", "member_type", "USER"),
				),
			},
			{
				ResourceName:      "graviteeam_protected_resource_member.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_protected_resource_member.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_protected_resource_member.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["protected_resource_id"] + "/" + rs.Primary.Attributes["member_id"] + "/" + rs.Primary.Attributes["member_type"] + "/" + rs.Primary.Attributes["role_id"], nil
				},
			},
		},
	})
}
