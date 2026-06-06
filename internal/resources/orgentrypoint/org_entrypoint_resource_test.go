package orgentrypoint_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgEntrypointResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_tag" "test" {
  name        = "test-acc-entrypoint-tag"
  description = "Acceptance test entrypoint tag"
}

resource "graviteeam_org_entrypoint" "test" {
  name        = "test-acc-entrypoint"
  description = "Acceptance test entrypoint"
  url         = "https://login.example.com"
  tags        = [graviteeam_org_tag.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_entrypoint.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_entrypoint.test", "name", "test-acc-entrypoint"),
					resource.TestCheckResourceAttr("graviteeam_org_entrypoint.test", "description", "Acceptance test entrypoint"),
					resource.TestCheckResourceAttr("graviteeam_org_entrypoint.test", "url", "https://login.example.com"),
					resource.TestCheckResourceAttr("graviteeam_org_entrypoint.test", "tags.#", "1"),
				),
			},
			{
				ResourceName:      "graviteeam_org_entrypoint.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_tag" "test" {
  name        = "test-acc-entrypoint-tag"
  description = "Acceptance test entrypoint tag"
}

resource "graviteeam_org_entrypoint" "test" {
  name        = "test-acc-entrypoint-updated"
  description = "Updated acceptance test entrypoint"
  url         = "https://login-updated.example.com"
  tags        = [graviteeam_org_tag.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_entrypoint.test", "name", "test-acc-entrypoint-updated"),
					resource.TestCheckResourceAttr("graviteeam_org_entrypoint.test", "description", "Updated acceptance test entrypoint"),
					resource.TestCheckResourceAttr("graviteeam_org_entrypoint.test", "url", "https://login-updated.example.com"),
				),
			},
		},
	})
}
