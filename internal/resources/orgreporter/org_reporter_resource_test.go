package orgreporter_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgReporterResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_reporter" "test" {
  name          = "test-acc-org-reporter"
  type          = "reporter-am-file"
  enabled       = true
  inherited     = false
  configuration = jsonencode({
    filename = "org-audit-test.log"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_reporter.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_reporter.test", "name", "test-acc-org-reporter"),
					resource.TestCheckResourceAttr("graviteeam_org_reporter.test", "type", "reporter-am-file"),
					resource.TestCheckResourceAttr("graviteeam_org_reporter.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_org_reporter.test", "inherited", "false"),
				),
			},
			{
				ResourceName:            "graviteeam_org_reporter.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_reporter" "test" {
  name          = "test-acc-org-reporter-updated"
  type          = "reporter-am-file"
  enabled       = true
  inherited     = false
  configuration = jsonencode({
    filename = "org-audit-test.log"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_reporter.test", "name", "test-acc-org-reporter-updated"),
				),
			},
		},
	})
}
