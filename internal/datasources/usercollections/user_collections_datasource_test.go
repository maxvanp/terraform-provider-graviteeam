package usercollections_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccUserCollectionDataSources_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-user-collections"
  description = "Domain for user collection data source test"

  oidc {}
  login_settings {}
}

resource "graviteeam_user" "test" {
  domain_id        = graviteeam_domain.test.id
  username         = "acctest-user-collections"
  email            = "acctest-user-collections@example.com"
  pre_registration = true
}

data "graviteeam_user_consents" "test" {
  domain_id = graviteeam_domain.test.id
  user_id   = graviteeam_user.test.id
}

data "graviteeam_user_credentials" "test" {
  domain_id = graviteeam_domain.test.id
  user_id   = graviteeam_user.test.id
}

data "graviteeam_user_devices" "test" {
  domain_id = graviteeam_domain.test.id
  user_id   = graviteeam_user.test.id
}

data "graviteeam_user_factors" "test" {
  domain_id = graviteeam_domain.test.id
  user_id   = graviteeam_user.test.id
}

data "graviteeam_user_identities" "test" {
  domain_id = graviteeam_domain.test.id
  user_id   = graviteeam_user.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.graviteeam_user_consents.test", "items_json", "[]"),
					resource.TestCheckResourceAttr("data.graviteeam_user_credentials.test", "items_json", "[]"),
					resource.TestCheckResourceAttr("data.graviteeam_user_devices.test", "items_json", "[]"),
					resource.TestCheckResourceAttr("data.graviteeam_user_factors.test", "items_json", "[]"),
					resource.TestCheckResourceAttr("data.graviteeam_user_identities.test", "items_json", "[]"),
				),
			},
		},
	})
}
