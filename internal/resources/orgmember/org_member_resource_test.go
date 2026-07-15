package orgmember_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgMemberResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_user" "test" {
  username         = "test-acc-org-member"
  password         = "SecurePass123!"
  email            = "test-acc-org-member@example.com"
  first_name       = "Org"
  last_name        = "Member"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_role" "test" {
  name            = "test-acc-org-member-role"
  description     = "Acceptance test org member role"
  assignable_type = "ORGANIZATION"
}

resource "graviteeam_org_member" "test" {
  member_id   = graviteeam_org_user.test.id
  member_type = "USER"
  role_id     = graviteeam_org_role.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_member.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_member.test", "member_type", "USER"),
				),
			},
			{
				ResourceName:      "graviteeam_org_member.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_org_member.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_org_member.test")
					}
					return rs.Primary.Attributes["member_id"] + "/" + rs.Primary.Attributes["member_type"] + "/" + rs.Primary.Attributes["role_id"], nil
				},
			},
		},
	})
}
