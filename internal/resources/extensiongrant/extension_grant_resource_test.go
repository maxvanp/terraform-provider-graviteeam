package extensiongrant_test

import (
	"fmt"
	"testing"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccExtensionGrantResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_extgrant" {
  name        = "test-extgrant-domain"
  description = "Domain for extension grant test"

  oidc {}
  login_settings {}
}

resource "graviteeam_extension_grant" "test" {
  domain_id     = graviteeam_domain.test_extgrant.id
  name          = "Test JWT Bearer Grant"
  type          = "jwtbearer-am-extension-grant"
  grant_type    = "urn:ietf:params:oauth:grant-type:jwt-bearer"
  configuration = jsonencode({
    publicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDummy"
  })
  create_user = false
  user_exists = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_extension_grant.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_extension_grant.test", "name", "Test JWT Bearer Grant"),
					resource.TestCheckResourceAttr("graviteeam_extension_grant.test", "type", "jwtbearer-am-extension-grant"),
					resource.TestCheckResourceAttr("graviteeam_extension_grant.test", "grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer"),
					resource.TestCheckResourceAttr("graviteeam_extension_grant.test", "create_user", "false"),
					resource.TestCheckResourceAttr("graviteeam_extension_grant.test", "user_exists", "true"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_extension_grant.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_extension_grant.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_extension_grant.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			// Update name
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_extgrant" {
  name        = "test-extgrant-domain"
  description = "Domain for extension grant test"

  oidc {}
  login_settings {}
}

resource "graviteeam_extension_grant" "test" {
  domain_id     = graviteeam_domain.test_extgrant.id
  name          = "Updated JWT Bearer Grant"
  type          = "jwtbearer-am-extension-grant"
  grant_type    = "urn:ietf:params:oauth:grant-type:jwt-bearer"
  configuration = jsonencode({
    publicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDummy"
  })
  create_user = false
  user_exists = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_extension_grant.test", "name", "Updated JWT Bearer Grant"),
				),
			},
		},
	})
}
