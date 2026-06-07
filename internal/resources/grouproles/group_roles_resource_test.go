package grouproles_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccGroupRolesResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-group-roles"
  oidc {}
  login_settings {}
}

resource "graviteeam_role" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "test-group-role"
  description = "Role for group roles test"
}

resource "graviteeam_group" "test" {
  domain_id = graviteeam_domain.test.id
  name      = "test-group-roles"
}

resource "graviteeam_group_roles" "test" {
  domain_id = graviteeam_domain.test.id
  group_id  = graviteeam_group.test.id
  roles     = [graviteeam_role.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_group_roles.test", "roles.#", "1"),
				),
			},
			{
				ResourceName:                         "graviteeam_group_roles.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "group_id",
				ImportStateVerifyIgnore:              []string{"roles"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_group_roles.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_group_roles.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["group_id"], nil
				},
			},
		},
	})
}
