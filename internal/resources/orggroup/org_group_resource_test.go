package orggroup_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgGroupResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_group" "test" {
  name        = "test-acc-org-group"
  description = "Acceptance test organization group"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_group.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "name", "test-acc-org-group"),
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "description", "Acceptance test organization group"),
				),
			},
			{
				ResourceName:      "graviteeam_org_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_group" "test" {
  name        = "test-acc-org-group-updated"
  description = "Updated organization group"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "name", "test-acc-org-group-updated"),
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "description", "Updated organization group"),
				),
			},
		},
	})
}
