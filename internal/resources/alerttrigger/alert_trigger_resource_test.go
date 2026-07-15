package alerttrigger_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccAlertTriggerResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-alert-trigger"
  description = "Domain for alert trigger test"

  oidc {}
  login_settings {}
}

resource "graviteeam_alert_trigger" "test" {
  domain_id = graviteeam_domain.test.id
  type      = "TOO_MANY_LOGIN_FAILURES"
  enabled   = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_alert_trigger.test", "id"),
					resource.TestCheckResourceAttrPair("graviteeam_alert_trigger.test", "domain_id", "graviteeam_domain.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_alert_trigger.test", "type", "TOO_MANY_LOGIN_FAILURES"),
					resource.TestCheckResourceAttr("graviteeam_alert_trigger.test", "enabled", "false"),
					resource.TestCheckResourceAttr("graviteeam_alert_trigger.test", "alert_notifier_ids.#", "0"),
				),
			},
			{
				ResourceName:      "graviteeam_alert_trigger.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_alert_trigger.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_alert_trigger.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["type"], nil
				},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-alert-trigger"
  description = "Domain for alert trigger test"

  oidc {}
  login_settings {}
}

resource "graviteeam_alert_trigger" "test" {
  domain_id = graviteeam_domain.test.id
  type      = "TOO_MANY_LOGIN_FAILURES"
  enabled   = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_alert_trigger.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_alert_trigger.test", "alert_notifier_ids.#", "0"),
				),
			},
		},
	})
}
