package applicationsecret_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccApplicationSecretResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-application-secret"
  oidc {}
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id = graviteeam_domain.test.id
  name      = "Test Application Secret"
  type      = "WEB"

  oauth_settings {
    redirect_uris  = ["https://example.com/auth"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
    scopes         = ["openid"]
  }
}

resource "graviteeam_application_secret" "test" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  name           = "test-acc-application-secret"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_application_secret.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_application_secret.test", "secret"),
					resource.TestCheckResourceAttr("graviteeam_application_secret.test", "name", "test-acc-application-secret"),
				),
			},
			{
				ResourceName:      "graviteeam_application_secret.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_application_secret.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_application_secret.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["application_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{
					"secret",
				},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-application-secret"
  oidc {}
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id = graviteeam_domain.test.id
  name      = "Test Application Secret"
  type      = "WEB"

  oauth_settings {
    redirect_uris  = ["https://example.com/auth"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
    scopes         = ["openid"]
  }
}

resource "graviteeam_application_secret" "test" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  name           = "test-acc-application-secret"
  renew_trigger  = "rotation-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_application_secret.test", "secret"),
					resource.TestCheckResourceAttr("graviteeam_application_secret.test", "renew_trigger", "rotation-1"),
				),
			},
		},
	})
}
