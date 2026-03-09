package applicationemail_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccApplicationEmailResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-app-email"
  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "test-app-email"
  type        = "WEB"
  oauth_settings {
    redirect_uris  = ["http://localhost:5000/callback"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
  }
}

resource "graviteeam_application_email" "test" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  template       = "REGISTRATION_CONFIRMATION"
  enabled        = true
  from           = "noreply@test.local"
  subject        = "Confirm"
  content        = "<html><body>Confirm your registration</body></html>"
  expires_after  = 86400
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_application_email.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_application_email.test", "template", "REGISTRATION_CONFIRMATION"),
					resource.TestCheckResourceAttr("graviteeam_application_email.test", "from", "noreply@test.local"),
				),
			},
			{
				ResourceName:      "graviteeam_application_email.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_application_email.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_application_email.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["application_id"] + "/" + rs.Primary.Attributes["template"], nil
				},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-app-email"
  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "test-app-email"
  type        = "WEB"
  oauth_settings {
    redirect_uris  = ["http://localhost:5000/callback"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
  }
}

resource "graviteeam_application_email" "test" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  template       = "REGISTRATION_CONFIRMATION"
  enabled        = true
  from           = "updated@test.local"
  subject        = "Updated confirm"
  content        = "<html><body>Updated confirm</body></html>"
  expires_after  = 86400
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_application_email.test", "from", "updated@test.local"),
					resource.TestCheckResourceAttr("graviteeam_application_email.test", "subject", "Updated confirm"),
				),
			},
		},
	})
}
