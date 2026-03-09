package botdetection_test

import (
	"fmt"
	"testing"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBotDetectionResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_botdet" {
  name        = "test-botdet-domain"
  description = "Domain for bot detection test"

  oidc {}
  login_settings {}
}

resource "graviteeam_bot_detection" "test" {
  domain_id      = graviteeam_domain.test_botdet.id
  name           = "Test reCAPTCHA"
  type           = "google-recaptcha-v3-am-bot-detection"
  detection_type = "CAPTCHA"
  configuration  = jsonencode({
    siteKey   = "6LeDummy_sitekey"
    secretKey = "6LeDummy_secretkey"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_bot_detection.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_bot_detection.test", "name", "Test reCAPTCHA"),
					resource.TestCheckResourceAttr("graviteeam_bot_detection.test", "type", "google-recaptcha-v3-am-bot-detection"),
					resource.TestCheckResourceAttr("graviteeam_bot_detection.test", "detection_type", "CAPTCHA"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_bot_detection.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_bot_detection.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_bot_detection.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			// Update name
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_botdet" {
  name        = "test-botdet-domain"
  description = "Domain for bot detection test"

  oidc {}
  login_settings {}
}

resource "graviteeam_bot_detection" "test" {
  domain_id      = graviteeam_domain.test_botdet.id
  name           = "Updated reCAPTCHA"
  type           = "google-recaptcha-v3-am-bot-detection"
  detection_type = "CAPTCHA"
  configuration  = jsonencode({
    siteKey   = "6LeDummy_sitekey"
    secretKey = "6LeDummy_secretkey"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_bot_detection.test", "name", "Updated reCAPTCHA"),
				),
			},
		},
	})
}
