package domain_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccDomain_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-domain"
  description = "Acceptance test domain"
  settings_json = jsonencode({
    tags = ["team-a"]
  })

  oidc {}
  login_settings {}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_domain.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "name", "test-acc-domain"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "description", "Acceptance test domain"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "enabled", "false"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "data_plane_id", "default"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "settings_json", `{"tags":["team-a"]}`),
					resource.TestCheckResourceAttrSet("graviteeam_domain.test", "default_idp_id"),
				),
			},
			// ImportState
			{
				ResourceName:            "graviteeam_domain.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"settings_json"},
			},
			// Update: add oidc, login_settings, enable
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-domain-updated"
  description = "Updated acceptance test domain"
  enabled     = true
  settings_json = jsonencode({
    tags = ["team-b", "iam"]
  })

  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }

  login_settings {
    register_enabled        = true
    forgot_password_enabled = true
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_domain.test", "name", "test-acc-domain-updated"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "description", "Updated acceptance test domain"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "settings_json", `{"tags":["team-b","iam"]}`),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "oidc.allow_localhost_redirect_uri", "true"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "oidc.allow_http_scheme_redirect_uri", "true"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "oidc.allow_wildcard_redirect_uri", "false"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "login_settings.register_enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "login_settings.forgot_password_enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_domain.test", "login_settings.identifier_first_enabled", "false"),
				),
			},
		},
	})
}
