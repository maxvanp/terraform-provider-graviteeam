package userrole_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccUserRoleResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-user-role"
  oidc {}
  login_settings {}
}

resource "graviteeam_role" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "test-user-role"
  description = "Role for user role test"
}

resource "graviteeam_user" "test" {
  domain_id  = graviteeam_domain.test.id
  username   = "test-user-role-user"
  email      = "user-role-test@example.com"
  first_name = "Test"
  last_name  = "User"
}

resource "graviteeam_user_role" "test" {
  domain_id = graviteeam_domain.test.id
  user_id   = graviteeam_user.test.id
  roles     = [graviteeam_role.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_user_role.test", "roles.#", "1"),
				),
			},
			{
				ResourceName:      "graviteeam_user_role.test",
				ImportState:       true,
				ImportStateVerify: false, // set resource has no id attribute, roles may differ in format
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_user_role.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_user_role.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["user_id"], nil
				},
			},
		},
	})
}
