package orgusertoken_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgUserTokenResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_user" "test" {
  username         = "test-acc-org-user-token"
  password         = "SecurePass123!"
  email            = "test-acc-org-user-token@example.com"
  first_name       = "Org"
  last_name        = "Token"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_user_token" "test" {
  user_id = graviteeam_org_user.test.id
  name    = "test-token"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_user_token.test", "id"),
					resource.TestCheckResourceAttrPair("graviteeam_org_user_token.test", "user_id", "graviteeam_org_user.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_org_user_token.test", "token_id"),
					resource.TestCheckResourceAttr("graviteeam_org_user_token.test", "name", "test-token"),
					resource.TestCheckResourceAttrSet("graviteeam_org_user_token.test", "token"),
				),
			},
			{
				ResourceName:      "graviteeam_org_user_token.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_org_user_token.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_org_user_token.test")
					}
					return rs.Primary.Attributes["user_id"] + "/" + rs.Primary.Attributes["token_id"], nil
				},
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}
