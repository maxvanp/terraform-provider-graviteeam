package orguser_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgUserResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_user" "test" {
  username         = "test-acc-org-user"
  password         = "SecurePass123!"
  email            = "test-acc-org-user@example.com"
  first_name       = "Org"
  last_name        = "User"
  enabled          = true
  pre_registration = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_user.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_user.test", "username", "test-acc-org-user"),
					resource.TestCheckResourceAttr("graviteeam_org_user.test", "email", "test-acc-org-user@example.com"),
					resource.TestCheckResourceAttr("graviteeam_org_user.test", "first_name", "Org"),
					resource.TestCheckResourceAttr("graviteeam_org_user.test", "last_name", "User"),
					resource.TestCheckResourceAttr("graviteeam_org_user.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_org_user.test", "pre_registration", "true"),
				),
			},
			{
				ResourceName:      "graviteeam_org_user.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
				},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_user" "test" {
  username         = "test-acc-org-user"
  password         = "SecurePass123!"
  email            = "test-acc-org-user-updated@example.com"
  first_name       = "Updated"
  last_name        = "User"
  enabled          = true
  pre_registration = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_user.test", "email", "test-acc-org-user-updated@example.com"),
					resource.TestCheckResourceAttr("graviteeam_org_user.test", "first_name", "Updated"),
				),
			},
		},
	})
}
