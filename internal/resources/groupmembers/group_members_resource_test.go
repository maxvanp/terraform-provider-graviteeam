package groupmembers_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccGroupMembersResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-group-members"
  oidc {}
  login_settings {}
}

resource "graviteeam_user" "test" {
  domain_id  = graviteeam_domain.test.id
  username   = "test-group-member"
  email      = "group-member@test.local"
  first_name = "Test"
  last_name  = "Member"
}

resource "graviteeam_group" "test" {
  domain_id = graviteeam_domain.test.id
  name      = "test-group-members"
}

resource "graviteeam_group_members" "test" {
  domain_id = graviteeam_domain.test.id
  group_id  = graviteeam_group.test.id
  members   = [graviteeam_user.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_group_members.test", "members.#", "1"),
				),
			},
			{
				ResourceName:      "graviteeam_group_members.test",
				ImportState:       true,
				ImportStateVerify: false, // set resource has no id attribute
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_group_members.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_group_members.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["group_id"], nil
				},
			},
		},
	})
}
