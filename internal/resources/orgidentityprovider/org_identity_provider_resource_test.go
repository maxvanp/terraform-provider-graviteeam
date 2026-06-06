package orgidentityprovider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgIdentityProviderResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_identity_provider" "test" {
  name          = "test-acc-org-idp"
  type          = "inline-am-idp"
  configuration = jsonencode({})
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_identity_provider.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "name", "test-acc-org-idp"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "type", "inline-am-idp"),
				),
			},
			{
				ResourceName:            "graviteeam_org_identity_provider.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_identity_provider" "test" {
  name          = "test-acc-org-idp-updated"
  type          = "inline-am-idp"
  configuration = jsonencode({})
  domain_whitelist = ["example.com"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "name", "test-acc-org-idp-updated"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "domain_whitelist.0", "example.com"),
				),
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_identity_provider" "test" {
  name          = "test-acc-org-idp-updated"
  type          = "inline-am-idp"
  configuration = jsonencode({})
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("graviteeam_org_identity_provider.test", "domain_whitelist.0"),
				),
			},
		},
	})
}
