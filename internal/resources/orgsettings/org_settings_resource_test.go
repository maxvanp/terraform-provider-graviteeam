package orgsettings_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgSettingsResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_settings" "test" {
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_settings.test", "id"),
				),
			},
			{
				ResourceName:      "graviteeam_org_settings.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
