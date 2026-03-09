package form_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccForm_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with template=LOGIN and HTML content
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-form"
  description = "Acceptance test domain for form"

  oidc {}
  login_settings {}
}

resource "graviteeam_form" "test" {
  domain_id = graviteeam_domain.test.id
  template  = "LOGIN"
  enabled   = true
  content   = "<html><body><h1>Login</h1></body></html>"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_form.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_form.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_form.test", "template", "LOGIN"),
					resource.TestCheckResourceAttr("graviteeam_form.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_form.test", "content", "<html><body><h1>Login</h1></body></html>"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_form.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_form.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_form.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["template"], nil
				},
			},
			// Update content
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-form"
  description = "Acceptance test domain for form"

  oidc {}
  login_settings {}
}

resource "graviteeam_form" "test" {
  domain_id = graviteeam_domain.test.id
  template  = "LOGIN"
  enabled   = true
  content   = "<html><body><h1>Welcome</h1><p>Please sign in</p></body></html>"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_form.test", "template", "LOGIN"),
					resource.TestCheckResourceAttr("graviteeam_form.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_form.test", "content", "<html><body><h1>Welcome</h1><p>Please sign in</p></body></html>"),
				),
			},
		},
	})
}
