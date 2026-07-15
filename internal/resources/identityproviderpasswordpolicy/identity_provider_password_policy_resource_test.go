package identityproviderpasswordpolicy_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccIdentityProviderPasswordPolicyResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-idp-password-policy"
  description = "Domain for identity provider password policy test"

  oidc {}
  login_settings {}
}

resource "graviteeam_password_policy" "first" {
  domain_id  = graviteeam_domain.test.id
  name       = "First IdP Password Policy"
  min_length = 10
}

resource "graviteeam_identity_provider_password_policy" "test" {
  domain_id            = graviteeam_domain.test.id
  identity_provider_id = graviteeam_domain.test.default_idp_id
  password_policy_id   = graviteeam_password_policy.first.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_identity_provider_password_policy.test", "id"),
					resource.TestCheckResourceAttrPair("graviteeam_identity_provider_password_policy.test", "domain_id", "graviteeam_domain.test", "id"),
					resource.TestCheckResourceAttrPair("graviteeam_identity_provider_password_policy.test", "identity_provider_id", "graviteeam_domain.test", "default_idp_id"),
					resource.TestCheckResourceAttrPair("graviteeam_identity_provider_password_policy.test", "password_policy_id", "graviteeam_password_policy.first", "id"),
				),
			},
			{
				ResourceName:      "graviteeam_identity_provider_password_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_identity_provider_password_policy.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_identity_provider_password_policy.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["identity_provider_id"], nil
				},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-idp-password-policy"
  description = "Domain for identity provider password policy test"

  oidc {}
  login_settings {}
}

resource "graviteeam_password_policy" "first" {
  domain_id  = graviteeam_domain.test.id
  name       = "First IdP Password Policy"
  min_length = 10
}

resource "graviteeam_password_policy" "second" {
  domain_id  = graviteeam_domain.test.id
  name       = "Second IdP Password Policy"
  min_length = 12
}

resource "graviteeam_identity_provider_password_policy" "test" {
  domain_id            = graviteeam_domain.test.id
  identity_provider_id = graviteeam_domain.test.default_idp_id
  password_policy_id   = graviteeam_password_policy.second.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("graviteeam_identity_provider_password_policy.test", "password_policy_id", "graviteeam_password_policy.second", "id"),
				),
			},
		},
	})
}
