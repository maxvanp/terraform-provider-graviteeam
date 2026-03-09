package analytics_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccAnalyticsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-ds-analytics"

  oidc {}
  login_settings {}
}

data "graviteeam_analytics" "test" {
  domain_id = graviteeam_domain.test.id
  type      = "GROUP_BY"
  field     = "application"
  interval  = 86400000
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_analytics.test", "result"),
					resource.TestCheckResourceAttrSet("data.graviteeam_analytics.test", "domain_id"),
					resource.TestCheckResourceAttr("data.graviteeam_analytics.test", "type", "GROUP_BY"),
				),
			},
		},
	})
}

func TestAccAnalyticsDataSource_withField(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-ds-analytics-field"

  oidc {}
  login_settings {}
}

data "graviteeam_analytics" "test" {
  domain_id = graviteeam_domain.test.id
  type      = "GROUP_BY"
  field     = "application"
  interval  = 86400000
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_analytics.test", "result"),
					resource.TestCheckResourceAttr("data.graviteeam_analytics.test", "type", "GROUP_BY"),
					resource.TestCheckResourceAttr("data.graviteeam_analytics.test", "field", "application"),
				),
			},
		},
	})
}
