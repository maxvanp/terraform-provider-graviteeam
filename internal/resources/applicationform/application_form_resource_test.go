package applicationform_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccApplicationFormResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-app-form"
  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "test-app-form"
  type        = "WEB"
  oauth_settings {
    redirect_uris  = ["http://localhost:5000/callback"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
  }
}

resource "graviteeam_application_form" "test" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  template       = "LOGIN"
  enabled        = true
  content        = "<html><body><h1>Custom Login</h1></body></html>"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_application_form.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_application_form.test", "template", "LOGIN"),
					resource.TestCheckResourceAttr("graviteeam_application_form.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "graviteeam_application_form.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_application_form.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_application_form.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["application_id"] + "/" + rs.Primary.Attributes["template"], nil
				},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-app-form"
  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "test-app-form"
  type        = "WEB"
  oauth_settings {
    redirect_uris  = ["http://localhost:5000/callback"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
  }
}

resource "graviteeam_application_form" "test" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  template       = "LOGIN"
  enabled        = true
  content        = "<html><body><h1>Updated Login</h1></body></html>"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_application_form.test", "content", "<html><body><h1>Updated Login</h1></body></html>"),
				),
			},
		},
	})
}
