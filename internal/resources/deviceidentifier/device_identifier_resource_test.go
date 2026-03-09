package deviceidentifier_test

import (
	"fmt"
	"testing"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDeviceIdentifierResource_basic(t *testing.T) {
	t.Skip("fingerprintjs-v3-am-device-identifier plugin not deployed in test environment")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_devid" {
  name        = "test-devid-domain"
  description = "Domain for device identifier test"

  oidc {}
  login_settings {}
}

resource "graviteeam_device_identifier" "test" {
  domain_id     = graviteeam_domain.test_devid.id
  name          = "Test FingerprintJS"
  type          = "fingerprintjs-v3-am-device-identifier"
  configuration = jsonencode({})
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_device_identifier.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_device_identifier.test", "name", "Test FingerprintJS"),
					resource.TestCheckResourceAttr("graviteeam_device_identifier.test", "type", "fingerprintjs-v3-am-device-identifier"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_device_identifier.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_device_identifier.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_device_identifier.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			// Update name
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_devid" {
  name        = "test-devid-domain"
  description = "Domain for device identifier test"

  oidc {}
  login_settings {}
}

resource "graviteeam_device_identifier" "test" {
  domain_id     = graviteeam_domain.test_devid.id
  name          = "Updated FingerprintJS"
  type          = "fingerprintjs-v3-am-device-identifier"
  configuration = jsonencode({})
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_device_identifier.test", "name", "Updated FingerprintJS"),
				),
			},
		},
	})
}
