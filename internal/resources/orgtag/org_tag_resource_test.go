package orgtag_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgTagResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_tag" "test" {
  name        = "test-acc-tag"
  description = "Acceptance test tag"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_tag.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_tag.test", "name", "test-acc-tag"),
					resource.TestCheckResourceAttr("graviteeam_org_tag.test", "description", "Acceptance test tag"),
				),
			},
			{
				ResourceName:      "graviteeam_org_tag.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_tag" "test" {
  name        = "test-acc-tag-updated"
  description = "Updated tag"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_tag.test", "name", "test-acc-tag-updated"),
					resource.TestCheckResourceAttr("graviteeam_org_tag.test", "description", "Updated tag"),
				),
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_tag" "test" {
  name = "test-acc-tag-updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_tag.test", "name", "test-acc-tag-updated"),
					resource.TestCheckNoResourceAttr("graviteeam_org_tag.test", "description"),
				),
			},
		},
	})
}
