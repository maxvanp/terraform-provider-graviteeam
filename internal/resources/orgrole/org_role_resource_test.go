package orgrole_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgRoleResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_role" "test" {
  name            = "test-acc-org-role"
  description     = "Acceptance test org role"
  assignable_type = "DOMAIN"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_role.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_role.test", "name", "test-acc-org-role"),
					resource.TestCheckResourceAttr("graviteeam_org_role.test", "assignable_type", "DOMAIN"),
				),
			},
			{
				ResourceName:      "graviteeam_org_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_role" "test" {
  name            = "test-acc-org-role-updated"
  description     = "Updated org role"
  assignable_type = "DOMAIN"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_role.test", "name", "test-acc-org-role-updated"),
					resource.TestCheckResourceAttr("graviteeam_org_role.test", "description", "Updated org role"),
				),
			},
		},
	})
}
