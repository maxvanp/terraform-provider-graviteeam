package applicationflow_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccApplicationFlowResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-app-flow"
  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "test-app-flow"
  type        = "WEB"
  oauth_settings {
    redirect_uris  = ["http://localhost:5000/callback"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
  }
}

resource "graviteeam_application_flow" "test" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  flows          = jsonencode([])
}
`,
				ExpectNonEmptyPlan: true, // API returns default flows even when we PUT []
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_application_flow.test", "flows"),
				),
			},
			{
				ResourceName:                         "graviteeam_application_flow.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "application_id",
				ImportStateVerifyIgnore:              []string{"flows"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_application_flow.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_application_flow.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["application_id"], nil
				},
			},
		},
	})
}
