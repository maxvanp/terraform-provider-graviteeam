package orggroupmembers_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgGroupMembersResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_user" "test" {
  username         = "test-acc-org-group-member"
  password         = "SecurePass123!"
  email            = "test-acc-org-group-member@example.com"
  first_name       = "Org"
  last_name        = "GroupMember"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_group" "test" {
  name = "test-acc-org-group-members"
}

resource "graviteeam_org_group_members" "test" {
  group_id = graviteeam_org_group.test.id
  members  = [graviteeam_org_user.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_group_members.test", "members.#", "1"),
				),
			},
			{
				ResourceName:                         "graviteeam_org_group_members.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "group_id",
				ImportStateVerifyIgnore:              []string{"members"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_org_group_members.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_org_group_members.test")
					}
					return rs.Primary.Attributes["group_id"], nil
				},
			},
		},
	})
}
