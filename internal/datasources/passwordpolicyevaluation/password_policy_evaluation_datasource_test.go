package passwordpolicyevaluation_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccPasswordPolicyEvaluationDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-password-policy-evaluation"
  description = "Domain for password policy evaluation data source test"

  oidc {}
  login_settings {}
}

resource "graviteeam_password_policy" "test" {
  domain_id  = graviteeam_domain.test.id
  name       = "Evaluation Password Policy"
  min_length = 8
}

data "graviteeam_password_policy_evaluation" "test" {
  domain_id = graviteeam_domain.test.id
  policy_id = graviteeam_password_policy.test.id
  password  = "SecurePass123!"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_password_policy_evaluation.test", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_password_policy_evaluation.test", "policy_id"),
				),
			},
		},
	})
}
