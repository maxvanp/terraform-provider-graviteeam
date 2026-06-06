package adminmetadata_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccAdminMetadataDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-admin-metadata"
  description = "Domain for admin metadata data source test"

  oidc {}
  login_settings {}
}

resource "graviteeam_user" "test" {
  domain_id        = graviteeam_domain.test.id
  username         = "admin-metadata-user"
  email            = "admin-metadata@example.com"
  first_name       = "Admin"
  last_name        = "Metadata"
  pre_registration = true
}

data "graviteeam_admin_metadata" "organization_audits" {
  kind = "organization_audits"
  size = 3
}

data "graviteeam_admin_metadata" "organization_environments" {
  kind = "organization_environments"
}

data "graviteeam_admin_metadata" "domain_by_hrid" {
  kind = "domain_by_hrid"
  hrid = graviteeam_domain.test.name
}

data "graviteeam_admin_metadata" "user_audits" {
  kind      = "user_audits"
  domain_id = graviteeam_domain.test.id
  user_id   = graviteeam_user.test.id
  size      = 3
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_admin_metadata.organization_audits", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_admin_metadata.organization_environments", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_admin_metadata.domain_by_hrid", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_admin_metadata.user_audits", "result_json"),
					resource.TestCheckResourceAttr("data.graviteeam_admin_metadata.organization_audits", "size", "3"),
					resource.TestCheckResourceAttr("data.graviteeam_admin_metadata.user_audits", "size", "3"),
				),
			},
		},
	})
}
