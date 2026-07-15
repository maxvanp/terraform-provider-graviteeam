package trustdomain_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccTrustDomain_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: trustDomainConfig("Terraform workload trust", 300, `["RS256"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_trust_domain.test", "id"),
					resource.TestCheckResourceAttrSet("graviteeam_trust_domain.test", "created_at"),
					resource.TestCheckResourceAttr("graviteeam_trust_domain.test", "name", "terraform.example.com"),
					resource.TestCheckResourceAttr("graviteeam_trust_domain.test", "bundle_source", "JWKS_URL"),
					resource.TestCheckResourceAttr("graviteeam_trust_domain.test", "refresh_interval_seconds", "300"),
					resource.TestCheckResourceAttr("graviteeam_trust_domain.test", "allowed_algorithms.0", "RS256"),
				),
			},
			{
				ResourceName:      "graviteeam_trust_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_trust_domain.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_trust_domain.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
			{
				Config: trustDomainConfig("Updated workload trust", 600, `["RS256", "ES256"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_trust_domain.test", "description", "Updated workload trust"),
					resource.TestCheckResourceAttr("graviteeam_trust_domain.test", "refresh_interval_seconds", "600"),
					resource.TestCheckResourceAttr("graviteeam_trust_domain.test", "allowed_algorithms.#", "2"),
				),
			},
		},
	})
}

func trustDomainConfig(description string, refresh int, algorithms string) string {
	return acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_domain" "test" {
  name        = "test-acc-trust-domain"
  description = "Domain for trust-domain acceptance test"

  settings_json = jsonencode({
    oidc = {
      workloadIdentitySettings = {
        enabled = true
      }
    }
  })

  oidc {}
  login_settings {}
}

resource "graviteeam_trust_domain" "test" {
  domain_id               = graviteeam_domain.test.id
  name                    = "terraform.example.com"
  description             = %q
  bundle_source           = "JWKS_URL"
  jwks_url                = "https://www.googleapis.com/oauth2/v3/certs"
  refresh_interval_seconds = %d
  allowed_algorithms      = %s
}
`, description, refresh, algorithms)
}
