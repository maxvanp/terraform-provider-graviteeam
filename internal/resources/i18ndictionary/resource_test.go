package i18ndictionary_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccI18nDictionary_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with name, locale=fr, entries map
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-i18n"
  description = "Acceptance test domain for i18n dictionary"

  oidc {}
  login_settings {}
}

resource "graviteeam_i18n_dictionary" "test" {
  domain_id = graviteeam_domain.test.id
  name      = "French"
  locale    = "fr"
  entries = {
    "login.title"    = "Connexion"
    "login.username" = "Identifiant"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_i18n_dictionary.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_i18n_dictionary.test", "domain_id"),
					resource.TestCheckResourceAttr("graviteeam_i18n_dictionary.test", "name", "French"),
					resource.TestCheckResourceAttr("graviteeam_i18n_dictionary.test", "locale", "fr"),
					resource.TestCheckResourceAttr("graviteeam_i18n_dictionary.test", "entries.login.title", "Connexion"),
					resource.TestCheckResourceAttr("graviteeam_i18n_dictionary.test", "entries.login.username", "Identifiant"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_i18n_dictionary.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_i18n_dictionary.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_i18n_dictionary.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			// Update name and add entries
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-i18n"
  description = "Acceptance test domain for i18n dictionary"

  oidc {}
  login_settings {}
}

resource "graviteeam_i18n_dictionary" "test" {
  domain_id = graviteeam_domain.test.id
  name      = "Francais"
  locale    = "fr"
  entries = {
    "login.title"    = "Connexion"
    "login.username" = "Identifiant"
    "login.password" = "Mot de passe"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_i18n_dictionary.test", "name", "Francais"),
					resource.TestCheckResourceAttr("graviteeam_i18n_dictionary.test", "entries.login.title", "Connexion"),
					resource.TestCheckResourceAttr("graviteeam_i18n_dictionary.test", "entries.login.username", "Identifiant"),
					resource.TestCheckResourceAttr("graviteeam_i18n_dictionary.test", "entries.login.password", "Mot de passe"),
				),
			},
		},
	})
}
