package domainflow_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccDomainFlowResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-domain-flow"
  oidc {}
  login_settings {}
}

resource "graviteeam_domain_flow" "test" {
  domain_id = graviteeam_domain.test.id
  flows     = jsonencode([])
}
`,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_domain_flow.test", "flows"),
				),
			},
			{
				ResourceName:                         "graviteeam_domain_flow.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "domain_id",
				ImportStateVerifyIgnore:              []string{"flows"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_domain_flow.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_domain_flow.test")
					}
					return rs.Primary.Attributes["domain_id"], nil
				},
			},
		},
	})
}
