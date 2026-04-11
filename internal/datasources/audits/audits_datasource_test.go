package audits_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccAuditsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-ds-audits"

  oidc {}
  login_settings {}
}

data "graviteeam_audits" "test" {
  domain_id = graviteeam_domain.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_audits.test", "audits"),
					resource.TestCheckResourceAttrSet("data.graviteeam_audits.test", "domain_id"),
					resource.TestCheckResourceAttr("data.graviteeam_audits.test", "size", "10"),
				),
			},
		},
	})
}

func TestAccAuditsDataSource_withSize(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-ds-audits-size"

  oidc {}
  login_settings {}
}

data "graviteeam_audits" "test" {
  domain_id = graviteeam_domain.test.id
  size      = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_audits.test", "audits"),
					resource.TestCheckResourceAttr("data.graviteeam_audits.test", "size", "5"),
				),
			},
		},
	})
}
