package passwordpolicy_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccPasswordPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-pwpolicy"
  description = "Domain for password policy acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_password_policy" "test" {
  domain_id  = graviteeam_domain.test.id
  name       = "Test Password Policy"
  min_length = 8
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_password_policy.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_password_policy.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_password_policy.test", "name", "Test Password Policy"),
					resource.TestCheckResourceAttr("graviteeam_password_policy.test", "min_length", "8"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_password_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_password_policy.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_password_policy.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			// Update: add include_numbers, letters_in_mixed_case, default_policy
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-pwpolicy"
  description = "Domain for password policy acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_password_policy" "test" {
  domain_id            = graviteeam_domain.test.id
  name                 = "Updated Password Policy"
  min_length           = 12
  include_numbers      = true
  letters_in_mixed_case = true
  default_policy       = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_password_policy.test", "name", "Updated Password Policy"),
					resource.TestCheckResourceAttr("graviteeam_password_policy.test", "min_length", "12"),
					resource.TestCheckResourceAttr("graviteeam_password_policy.test", "include_numbers", "true"),
					resource.TestCheckResourceAttr("graviteeam_password_policy.test", "letters_in_mixed_case", "true"),
					resource.TestCheckResourceAttr("graviteeam_password_policy.test", "default_policy", "true"),
				),
			},
		},
	})
}
