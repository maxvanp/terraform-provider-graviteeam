package group_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-group"
  description = "Domain for group acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_role" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Group Test Role"
  description = "Role for group test"
}

resource "graviteeam_user" "test" {
  domain_id        = graviteeam_domain.test.id
  username         = "group-test-user"
  email            = "grouptest@example.com"
  first_name       = "Group"
  last_name        = "Tester"
  pre_registration = true
}

resource "graviteeam_group" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Test Group"
  description = "Acceptance test group"
  roles       = [graviteeam_role.test.id]
  members     = [graviteeam_user.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_group.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_group.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_group.test", "name", "Test Group"),
					resource.TestCheckResourceAttr("graviteeam_group.test", "description", "Acceptance test group"),
					resource.TestCheckResourceAttr("graviteeam_group.test", "roles.#", "1"),
					resource.TestCheckResourceAttr("graviteeam_group.test", "members.#", "1"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_group.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_group.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_group.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"roles", "members"},
			},
			// Update: change name
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-group"
  description = "Domain for group acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_role" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Group Test Role"
  description = "Role for group test"
}

resource "graviteeam_user" "test" {
  domain_id        = graviteeam_domain.test.id
  username         = "group-test-user"
  email            = "grouptest@example.com"
  first_name       = "Group"
  last_name        = "Tester"
  pre_registration = true
}

resource "graviteeam_group" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Updated Test Group"
  description = "Updated acceptance test group"
  roles       = [graviteeam_role.test.id]
  members     = [graviteeam_user.test.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_group.test", "name", "Updated Test Group"),
					resource.TestCheckResourceAttr("graviteeam_group.test", "description", "Updated acceptance test group"),
					resource.TestCheckResourceAttr("graviteeam_group.test", "roles.#", "1"),
					resource.TestCheckResourceAttr("graviteeam_group.test", "members.#", "1"),
				),
			},
		},
	})
}
