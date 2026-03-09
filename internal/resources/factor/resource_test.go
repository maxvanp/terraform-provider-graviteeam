package factor_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccFactor_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-factor"
  description = "Domain for factor acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_factor" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Test TOTP Factor"
  factor_type = "TOTP"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_factor.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_factor.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_factor.test", "name", "Test TOTP Factor"),
					resource.TestCheckResourceAttr("graviteeam_factor.test", "factor_type", "TOTP"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_factor.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_factor.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_factor.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			// Update name
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-factor"
  description = "Domain for factor acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_factor" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Updated TOTP Factor"
  factor_type = "TOTP"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_factor.test", "name", "Updated TOTP Factor"),
					resource.TestCheckResourceAttr("graviteeam_factor.test", "factor_type", "TOTP"),
				),
			},
		},
	})
}
