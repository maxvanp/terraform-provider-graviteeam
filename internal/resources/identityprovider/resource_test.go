package identityprovider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccIdentityProvider_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-idp"
  description = "Domain for identity provider acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_identity_provider" "test" {
  domain_id     = graviteeam_domain.test.id
  name          = "Test Inline IdP"
  type          = "inline-am-idp"
  configuration = jsonencode({
    users = [
      {
        firstname = "Test"
        lastname  = "User"
        username  = "testuser"
        password  = "Password1!"
        email     = "testuser@example.com"
      }
    ]
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_identity_provider.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_identity_provider.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_identity_provider.test", "name", "Test Inline IdP"),
					resource.TestCheckResourceAttr("graviteeam_identity_provider.test", "type", "inline-am-idp"),
					resource.TestCheckResourceAttr("graviteeam_identity_provider.test", "external", "false"),
				),
			},
			// ImportState (ignore configuration because API masks sensitive fields)
			{
				ResourceName:      "graviteeam_identity_provider.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_identity_provider.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_identity_provider.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			// Update: change name and add mappers
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-idp"
  description = "Domain for identity provider acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_identity_provider" "test" {
  domain_id     = graviteeam_domain.test.id
  name          = "Updated Inline IdP"
  type          = "inline-am-idp"
  configuration = jsonencode({
    users = [
      {
        firstname = "Test"
        lastname  = "User"
        username  = "testuser"
        password  = "Password1!"
        email     = "testuser@example.com"
      }
    ]
  })
  mappers = {
    "email"    = "email"
    "username" = "username"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_identity_provider.test", "name", "Updated Inline IdP"),
					resource.TestCheckResourceAttr("graviteeam_identity_provider.test", "mappers.email", "email"),
					resource.TestCheckResourceAttr("graviteeam_identity_provider.test", "mappers.username", "username"),
				),
			},
		},
	})
}
