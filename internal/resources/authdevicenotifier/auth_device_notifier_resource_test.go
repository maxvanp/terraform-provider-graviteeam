package authdevicenotifier_test

import (
	"fmt"
	"testing"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAuthDeviceNotifierResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_authdevnotif" {
  name        = "test-authdevnotif-domain"
  description = "Domain for auth device notifier test"

  oidc {}
  login_settings {}
}

resource "graviteeam_auth_device_notifier" "test" {
  domain_id     = graviteeam_domain.test_authdevnotif.id
  name          = "Test HTTP Notifier"
  type          = "http-am-authdevice-notifier"
  configuration = jsonencode({
    endpoint    = "https://example.com/ciba/notify"
    headerName  = "Authorization"
    headerValue = "Bearer test-token"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_auth_device_notifier.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_auth_device_notifier.test", "name", "Test HTTP Notifier"),
					resource.TestCheckResourceAttr("graviteeam_auth_device_notifier.test", "type", "http-am-authdevice-notifier"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_auth_device_notifier.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_auth_device_notifier.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_auth_device_notifier.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			// Update name
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_authdevnotif" {
  name        = "test-authdevnotif-domain"
  description = "Domain for auth device notifier test"

  oidc {}
  login_settings {}
}

resource "graviteeam_auth_device_notifier" "test" {
  domain_id     = graviteeam_domain.test_authdevnotif.id
  name          = "Updated HTTP Notifier"
  type          = "http-am-authdevice-notifier"
  configuration = jsonencode({
    endpoint    = "https://example.com/ciba/notify"
    headerName  = "Authorization"
    headerValue = "Bearer test-token"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_auth_device_notifier.test", "name", "Updated HTTP Notifier"),
				),
			},
		},
	})
}
