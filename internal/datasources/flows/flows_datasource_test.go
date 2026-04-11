package flows_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccFlowsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-ds-flows"

  oidc {}
  login_settings {}
}

data "graviteeam_flows" "test" {
  domain_id = graviteeam_domain.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_flows.test", "flows"),
					resource.TestCheckResourceAttrSet("data.graviteeam_flows.test", "domain_id"),
				),
			},
		},
	})
}
