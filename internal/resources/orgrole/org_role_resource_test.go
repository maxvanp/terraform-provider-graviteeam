package orgrole_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgRoleResource_basic(t *testing.T) {
	name := fmt.Sprintf("test-acc-org-role-%d", time.Now().UnixNano())
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_org_role" "test" {
  name            = %q
  description     = "Acceptance test org role"
  assignable_type = "DOMAIN"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_role.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_role.test", "name", name),
					resource.TestCheckResourceAttr("graviteeam_org_role.test", "assignable_type", "DOMAIN"),
				),
			},
			{
				ResourceName:      "graviteeam_org_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_org_role" "test" {
  name            = %q
  description     = "Updated org role"
  assignable_type = "DOMAIN"
}
`, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_role.test", "name", updatedName),
					resource.TestCheckResourceAttr("graviteeam_org_role.test", "description", "Updated org role"),
				),
			},
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_org_role" "test" {
  name            = %q
  assignable_type = "DOMAIN"
}
`, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_role.test", "name", updatedName),
					resource.TestCheckNoResourceAttr("graviteeam_org_role.test", "description"),
					resource.TestCheckNoResourceAttr("graviteeam_org_role.test", "permissions.#"),
				),
			},
		},
	})
}
