package authorizationengine_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccAuthorizationEngineResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set, skipping acceptance test")
	}

	pluginType := deployedAuthorizationEnginePlugin(t)
	if pluginType == "" {
		t.Skip("no deployed authorization engine plugin in test environment")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_domain" "test_authz_engine" {
  name        = "test-authz-engine-domain"
  description = "Domain for authorization engine test"

  oidc {}
  login_settings {}
}

resource "graviteeam_authorization_engine" "test" {
  domain_id     = graviteeam_domain.test_authz_engine.id
  name          = "Test authorization engine"
  type          = %q
  configuration = jsonencode({})
}
`, pluginType),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_authorization_engine.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_authorization_engine.test", "name", "Test authorization engine"),
					resource.TestCheckResourceAttr("graviteeam_authorization_engine.test", "type", pluginType),
				),
			},
			{
				ResourceName:      "graviteeam_authorization_engine.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_authorization_engine.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_authorization_engine.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_domain" "test_authz_engine" {
  name        = "test-authz-engine-domain"
  description = "Domain for authorization engine test"

  oidc {}
  login_settings {}
}

resource "graviteeam_authorization_engine" "test" {
  domain_id     = graviteeam_domain.test_authz_engine.id
  name          = "Updated authorization engine"
  type          = %q
  configuration = jsonencode({})
}
`, pluginType),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_authorization_engine.test", "name", "Updated authorization engine"),
				),
			},
		},
	})
}

func deployedAuthorizationEnginePlugin(t *testing.T) string {
	t.Helper()

	token := managementToken(t)
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8093/management/platform/plugins/authorization-engines", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authorization engine plugin list returned status %d", resp.StatusCode)
	}

	var plugins []struct {
		ID       string `json:"id"`
		Deployed bool   `json:"deployed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		t.Fatal(err)
	}
	for _, plugin := range plugins {
		if plugin.Deployed {
			return plugin.ID
		}
	}
	return ""
}

func managementToken(t *testing.T) string {
	t.Helper()

	body := url.Values{}
	body.Set("grant_type", "client_credentials")
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8093/management/auth/token", strings.NewReader(body.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("admin", "adminadmin")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token endpoint returned status %d", resp.StatusCode)
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		t.Fatal(err)
	}
	return token.AccessToken
}
