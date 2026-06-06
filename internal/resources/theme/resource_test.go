package theme_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccTheme_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with color hex values
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-theme"
  description = "Acceptance test domain for theme"

  oidc {}
  login_settings {}
}

resource "graviteeam_theme" "test" {
  domain_id                  = graviteeam_domain.test.id
  logo_url                   = "https://example.com/logo.png"
  logo_width                 = 120
  favicon_url                = "https://example.com/favicon.ico"
  primary_button_color_hex   = "#4CAF50"
  secondary_button_color_hex = "#FF5722"
  primary_text_color_hex     = "#212121"
  secondary_text_color_hex   = "#757575"
  css                        = "body { color: #212121; }"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_theme.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_theme.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "logo_url", "https://example.com/logo.png"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "logo_width", "120"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "favicon_url", "https://example.com/favicon.ico"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "primary_button_color_hex", "#4CAF50"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "secondary_button_color_hex", "#FF5722"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "primary_text_color_hex", "#212121"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "secondary_text_color_hex", "#757575"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "css", "body { color: #212121; }"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_theme.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_theme.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_theme.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			// Update colors and add css
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-theme"
  description = "Acceptance test domain for theme"

  oidc {}
  login_settings {}
}

resource "graviteeam_theme" "test" {
  domain_id                  = graviteeam_domain.test.id
  primary_button_color_hex   = "#1976D2"
  secondary_button_color_hex = "#E91E63"
  primary_text_color_hex     = "#000000"
  secondary_text_color_hex   = "#616161"
  css                        = "body { font-family: Arial, sans-serif; }"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_theme.test", "primary_button_color_hex", "#1976D2"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "secondary_button_color_hex", "#E91E63"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "primary_text_color_hex", "#000000"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "secondary_text_color_hex", "#616161"),
					resource.TestCheckResourceAttr("graviteeam_theme.test", "css", "body { font-family: Arial, sans-serif; }"),
				),
			},
			// Update: remove optional fields to ensure empty values clear remote state
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-theme"
  description = "Acceptance test domain for theme"

  oidc {}
  login_settings {}
}

resource "graviteeam_theme" "test" {
  domain_id = graviteeam_domain.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("graviteeam_theme.test", "logo_url"),
					resource.TestCheckNoResourceAttr("graviteeam_theme.test", "logo_width"),
					resource.TestCheckNoResourceAttr("graviteeam_theme.test", "favicon_url"),
					resource.TestCheckNoResourceAttr("graviteeam_theme.test", "primary_button_color_hex"),
					resource.TestCheckNoResourceAttr("graviteeam_theme.test", "secondary_button_color_hex"),
					resource.TestCheckNoResourceAttr("graviteeam_theme.test", "primary_text_color_hex"),
					resource.TestCheckNoResourceAttr("graviteeam_theme.test", "secondary_text_color_hex"),
					resource.TestCheckNoResourceAttr("graviteeam_theme.test", "css"),
				),
			},
		},
	})
}
