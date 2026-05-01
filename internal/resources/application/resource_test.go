package application_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccApplication_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-app"
  description = "Domain for application acceptance test"

  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Test Application"
  type        = "WEB"
  description = "Acceptance test application"

  oauth_settings {
    redirect_uris  = ["http://localhost:5000/auth"]
    grant_types    = ["authorization_code", "refresh_token"]
    response_types = ["code"]
    scopes         = ["openid", "profile", "email"]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_application.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_application.test", "domain_id"),
					resource.TestCheckResourceAttrSet("graviteeam_application.test", "client_id"),
					resource.TestCheckResourceAttrSet("graviteeam_application.test", "client_secret"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "name", "Test Application"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "type", "WEB"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "description", "Acceptance test application"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "oauth_settings.redirect_uris.#", "1"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "oauth_settings.redirect_uris.0", "http://localhost:5000/auth"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "oauth_settings.grant_types.#", "2"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "oauth_settings.response_types.#", "1"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "oauth_settings.scopes.#", "3"),
				),
			},
			// ImportState (ignore client_secret since API masks it)
			{
				ResourceName:      "graviteeam_application.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_application.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_application.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
			// Update: add mfa_settings and more redirect_uris
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-app"
  description = "Domain for application acceptance test"

  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }
  login_settings {}
}

resource "graviteeam_factor" "totp" {
  domain_id   = graviteeam_domain.test.id
  name        = "App Test TOTP"
  factor_type = "TOTP"
}

resource "graviteeam_application" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Updated Test Application"
  type        = "WEB"
  description = "Updated acceptance test application"

  factors = [graviteeam_factor.totp.id]

  oauth_settings {
    redirect_uris  = ["http://localhost:5000/auth", "http://localhost:5000/callback"]
    grant_types    = ["authorization_code", "refresh_token"]
    response_types = ["code"]
    scopes         = ["openid", "profile", "email"]
  }

  mfa_settings {
    enrollment = "OPTIONAL"
    challenge  = "REQUIRED"
  }

  settings_json = jsonencode({
    advanced = {
      skipConsent = true
    }
    oauth = {
      forcePKCE               = true
      tokenEndpointAuthMethod = "client_secret_post"
      tokenCustomClaims = [
        {
          claimName  = "tenant"
          claimValue = "terraform"
          tokenType  = "ACCESS_TOKEN"
        }
      ]
    }
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_application.test", "name", "Updated Test Application"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "description", "Updated acceptance test application"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "oauth_settings.redirect_uris.#", "2"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "oauth_settings.redirect_uris.1", "http://localhost:5000/callback"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "factors.#", "1"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "mfa_settings.enrollment", "OPTIONAL"),
					resource.TestCheckResourceAttr("graviteeam_application.test", "mfa_settings.challenge", "REQUIRED"),
				),
			},
		},
	})
}
