package applications_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccApplicationsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-applications-search"
  oidc {}
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id = graviteeam_domain.test.id
  name      = "Searchable Application"
  type      = "SERVICE"
}

data "graviteeam_applications" "test" {
  domain_id         = graviteeam_domain.test.id
  query             = "Searchable"
  application_types = ["SERVICE"]
  limit             = 10
  depends_on        = [graviteeam_application.test]
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.graviteeam_applications.test", "pagination_mode", "page"),
				resource.TestCheckResourceAttr("data.graviteeam_applications.test", "limit", "10"),
				resource.TestCheckResourceAttrSet("data.graviteeam_applications.test", "result_json"),
			),
		}},
	})
}
