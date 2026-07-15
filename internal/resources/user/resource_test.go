package user_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccUser_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-user"
  description = "Domain for user acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_user" "test" {
  domain_id        = graviteeam_domain.test.id
  username         = "acctest-user"
  email            = "acctest@example.com"
  first_name       = "Acc"
  last_name        = "Test"
  display_name     = "Acc Test"
  force_reset_password = false
  pre_registration = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_user.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_user.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "username", "acctest-user"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "email", "acctest@example.com"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "first_name", "Acc"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "last_name", "Test"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "display_name", "Acc Test"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "force_reset_password", "false"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "enabled", "false"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "locked", "false"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "pre_registration", "true"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_user.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_user.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_user.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			// Update: change email and names
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-user"
  description = "Domain for user acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_user" "test" {
  domain_id        = graviteeam_domain.test.id
  username         = "acctest-user-updated"
  email            = "updated@example.com"
  first_name       = "Updated"
  last_name        = "User"
  display_name     = "Updated User"
  force_reset_password = true
  enabled          = true
  locked           = true
  pre_registration = true
  registration_confirmation_trigger = "confirmation-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_user.test", "username", "acctest-user-updated"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "email", "updated@example.com"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "first_name", "Updated"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "last_name", "User"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "display_name", "Updated User"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "force_reset_password", "true"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "locked", "true"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "registration_confirmation_trigger", "confirmation-1"),
				),
			},
			// Update: unlock through the dedicated endpoint
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-user"
  description = "Domain for user acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_user" "test" {
  domain_id        = graviteeam_domain.test.id
  username         = "acctest-user-updated"
  email            = "updated@example.com"
  first_name       = "Updated"
  last_name        = "User"
  display_name     = "Updated User"
  force_reset_password = true
  enabled          = true
  locked           = false
  pre_registration = true
  reset_password   = "SecurePass123!"
  reset_password_trigger = "reset-1"
  registration_confirmation_trigger = "confirmation-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_user.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "locked", "false"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "reset_password_trigger", "reset-1"),
					resource.TestCheckResourceAttr("graviteeam_user.test", "registration_confirmation_trigger", "confirmation-1"),
				),
			},
		},
	})
}
