package scope_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccScope_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-scope"
  description = "Domain for scope acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_scope" "test" {
  domain_id   = graviteeam_domain.test.id
  key         = "test_scope"
  name        = "Test Scope"
  description = "A test scope"
  expires_in  = 3600
  icon_uri    = "https://example.com/icon.svg"
  parameterized = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_scope.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_scope.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "key", "test_scope"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "name", "Test Scope"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "description", "A test scope"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "discovery", "true"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "expires_in", "3600"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "icon_uri", "https://example.com/icon.svg"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "parameterized", "true"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_scope.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_scope.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_scope.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			// Update name and discovery
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-scope"
  description = "Domain for scope acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_scope" "test" {
  domain_id   = graviteeam_domain.test.id
  key         = "test_scope"
  name        = "Updated Test Scope"
  description = "A test scope"
  discovery   = false
  icon_uri    = "https://example.com/updated-icon.svg"
  parameterized = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_scope.test", "name", "Updated Test Scope"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "discovery", "false"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "icon_uri", "https://example.com/updated-icon.svg"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "parameterized", "true"),
				),
			},
			// Update: remove optional fields to ensure empty values clear remote state
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-scope"
  description = "Domain for scope acceptance test"

  oidc {}
  login_settings {}
}

resource "graviteeam_scope" "test" {
  domain_id = graviteeam_domain.test.id
  key       = "test_scope"
  name      = "Updated Test Scope"
  discovery = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_scope.test", "name", "Updated Test Scope"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "discovery", "false"),
					resource.TestCheckNoResourceAttr("graviteeam_scope.test", "description"),
					resource.TestCheckNoResourceAttr("graviteeam_scope.test", "expires_in"),
					resource.TestCheckNoResourceAttr("graviteeam_scope.test", "icon_uri"),
					resource.TestCheckResourceAttr("graviteeam_scope.test", "parameterized", "false"),
				),
			},
		},
	})
}
