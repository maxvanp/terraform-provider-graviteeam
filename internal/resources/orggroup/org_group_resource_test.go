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
resource "graviteeam_org_role" "test" {
  name            = "test-acc-org-group-role"
  assignable_type = "ORGANIZATION"
}

resource "graviteeam_org_group" "test" {
  name        = "test-acc-org-group"
  description = "Acceptance test organization group"
  roles       = [graviteeam_org_role.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_group.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "name", "test-acc-org-group"),
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "description", "Acceptance test organization group"),
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "roles.#", "1"),
				),
			},
			{
				ResourceName:      "graviteeam_org_group.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"roles",
				},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_role" "test" {
  name            = "test-acc-org-group-role"
  assignable_type = "ORGANIZATION"
}

resource "graviteeam_org_group" "test" {
  name        = "test-acc-org-group-updated"
  description = "Updated organization group"
  roles       = [graviteeam_org_role.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "name", "test-acc-org-group-updated"),
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "description", "Updated organization group"),
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "roles.#", "1"),
				),
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_role" "test" {
  name            = "test-acc-org-group-role"
  assignable_type = "ORGANIZATION"
}

resource "graviteeam_org_group" "test" {
  name = "test-acc-org-group-updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_group.test", "name", "test-acc-org-group-updated"),
					resource.TestCheckNoResourceAttr("graviteeam_org_group.test", "description"),
					resource.TestCheckNoResourceAttr("graviteeam_org_group.test", "roles.#"),
				),
			},
		},
	})
}
