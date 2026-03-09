package reporter_test

import (
	"fmt"
	"testing"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccReporterResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_reporter" {
  name        = "test-reporter-domain"
  description = "Domain for reporter test"

  oidc {}
  login_settings {}
}

resource "graviteeam_reporter" "test" {
  domain_id     = graviteeam_domain.test_reporter.id
  name          = "Test File Reporter"
  type          = "reporter-am-file"
  enabled       = true
  configuration = jsonencode({
    filename = "audit-test.log"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_reporter.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_reporter.test", "name", "Test File Reporter"),
					resource.TestCheckResourceAttr("graviteeam_reporter.test", "type", "reporter-am-file"),
					resource.TestCheckResourceAttr("graviteeam_reporter.test", "enabled", "true"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_reporter.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_reporter.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_reporter.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			// Update name
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_reporter" {
  name        = "test-reporter-domain"
  description = "Domain for reporter test"

  oidc {}
  login_settings {}
}

resource "graviteeam_reporter" "test" {
  domain_id     = graviteeam_domain.test_reporter.id
  name          = "Updated File Reporter"
  type          = "reporter-am-file"
  enabled       = true
  configuration = jsonencode({
    filename = "audit-test.log"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_reporter.test", "name", "Updated File Reporter"),
				),
			},
		},
	})
}
