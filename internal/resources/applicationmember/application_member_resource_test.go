package applicationmember_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccApplicationMemberResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name = "test-acc-application-member"
  oidc {}
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id = graviteeam_domain.test.id
  name      = "Test Application Member"
  type      = "WEB"

  oauth_settings {
    redirect_uris  = ["https://example.com/auth"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
    scopes         = ["openid"]
  }
}

resource "graviteeam_org_user" "test" {
  username         = "test-acc-application-member"
  password         = "SecurePass123!"
  email            = "test-acc-application-member@example.com"
  first_name       = "Application"
  last_name        = "Member"
  enabled          = true
  pre_registration = true
}

resource "graviteeam_org_role" "test" {
  name            = "test-acc-application-member-role"
  description     = "Acceptance test application member role"
  assignable_type = "APPLICATION"
}

resource "graviteeam_application_member" "test" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  member_id      = graviteeam_org_user.test.id
  member_type    = "USER"
  role_id        = graviteeam_org_role.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_application_member.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_application_member.test", "member_type", "USER"),
				),
			},
			{
				ResourceName:      "graviteeam_application_member.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_application_member.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_application_member.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["application_id"] + "/" + rs.Primary.Attributes["member_id"] + "/" + rs.Primary.Attributes["member_type"] + "/" + rs.Primary.Attributes["role_id"], nil
				},
			},
		},
	})
}
