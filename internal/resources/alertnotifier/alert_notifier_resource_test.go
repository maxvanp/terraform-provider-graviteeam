package alertnotifier_test

import (
	"fmt"
	"testing"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAlertNotifierResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_alertnotif" {
  name        = "test-alertnotif-domain"
  description = "Domain for alert notifier test"

  oidc {}
  login_settings {}
}

resource "graviteeam_alert_notifier" "test" {
  domain_id     = graviteeam_domain.test_alertnotif.id
  name          = "Test Webhook Notifier"
  type          = "webhook-notifier"
  enabled       = true
  configuration = jsonencode({
    url    = "https://hooks.slack.example.com/test"
    method = "POST"
    body   = "{\"text\": \"Alert: {{alert.name}}\"}"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_alert_notifier.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_alert_notifier.test", "name", "Test Webhook Notifier"),
					resource.TestCheckResourceAttr("graviteeam_alert_notifier.test", "type", "webhook-notifier"),
					resource.TestCheckResourceAttr("graviteeam_alert_notifier.test", "enabled", "true"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_alert_notifier.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_alert_notifier.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_alert_notifier.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			// Update name and enabled
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_alertnotif" {
  name        = "test-alertnotif-domain"
  description = "Domain for alert notifier test"

  oidc {}
  login_settings {}
}

resource "graviteeam_alert_notifier" "test" {
  domain_id     = graviteeam_domain.test_alertnotif.id
  name          = "Updated Webhook Notifier"
  type          = "webhook-notifier"
  enabled       = false
  configuration = jsonencode({
    url    = "https://hooks.slack.example.com/test"
    method = "POST"
    body   = "{\"text\": \"Alert: {{alert.name}}\"}"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_alert_notifier.test", "name", "Updated Webhook Notifier"),
					resource.TestCheckResourceAttr("graviteeam_alert_notifier.test", "enabled", "false"),
				),
			},
		},
	})
}
