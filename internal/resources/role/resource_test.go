package role_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccRole_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-role"
  description = "Domain for role acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_role" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Test Role"
  description = "A test role"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_role.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_role.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_role.test", "name", "Test Role"),
					resource.TestCheckResourceAttr("graviteeam_role.test", "description", "A test role"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_role.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_role.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_role.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			// Update: add oauth_scopes
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-role"
  description = "Domain for role acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_scope" "role_scope" {
  domain_id   = graviteeam_domain.test.id
  key         = "test-role-scope"
  name        = "Test Role Scope"
  description = "Scope for role testing"
}

resource "graviteeam_role" "test" {
  domain_id    = graviteeam_domain.test.id
  name         = "Updated Test Role"
  description  = "An updated test role"
  oauth_scopes = [graviteeam_scope.role_scope.key]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_role.test", "name", "Updated Test Role"),
					resource.TestCheckResourceAttr("graviteeam_role.test", "description", "An updated test role"),
					resource.TestCheckResourceAttr("graviteeam_role.test", "oauth_scopes.#", "1"),
					resource.TestCheckResourceAttr("graviteeam_role.test", "oauth_scopes.0", "test-role-scope"),
				),
			},
		},
	})
}
