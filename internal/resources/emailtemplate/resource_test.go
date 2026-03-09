package emailtemplate_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccEmailTemplate_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with REGISTRATION_CONFIRMATION
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-emailtpl"
  description = "Acceptance test domain for email template"

  oidc {}
  login_settings {}
}

resource "graviteeam_email_template" "test" {
  domain_id     = graviteeam_domain.test.id
  template      = "REGISTRATION_CONFIRMATION"
  enabled       = true
  from          = "noreply@test.local"
  subject       = "Confirm your registration"
  content       = "<html><body><h1>Welcome</h1><p>Please confirm your email.</p></body></html>"
  expires_after = 86400
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_email_template.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_email_template.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_email_template.test", "template", "REGISTRATION_CONFIRMATION"),
					resource.TestCheckResourceAttr("graviteeam_email_template.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_email_template.test", "from", "noreply@test.local"),
					resource.TestCheckResourceAttr("graviteeam_email_template.test", "subject", "Confirm your registration"),
					resource.TestCheckResourceAttr("graviteeam_email_template.test", "content", "<html><body><h1>Welcome</h1><p>Please confirm your email.</p></body></html>"),
					resource.TestCheckResourceAttr("graviteeam_email_template.test", "expires_after", "86400"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_email_template.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_email_template.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_email_template.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["template"], nil
				},
			},
			// Update from and subject
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-emailtpl"
  description = "Acceptance test domain for email template"

  oidc {}
  login_settings {}
}

resource "graviteeam_email_template" "test" {
  domain_id     = graviteeam_domain.test.id
  template      = "REGISTRATION_CONFIRMATION"
  enabled       = true
  from          = "support@test.local"
  subject       = "Please confirm your account"
  content       = "<html><body><h1>Welcome</h1><p>Please confirm your email.</p></body></html>"
  expires_after = 86400
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_email_template.test", "from", "support@test.local"),
					resource.TestCheckResourceAttr("graviteeam_email_template.test", "subject", "Please confirm your account"),
				),
			},
		},
	})
}
