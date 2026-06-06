package orgform_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgFormResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_form" "test" {
  template = "LOGIN"
  enabled  = true
  content  = "<html><body><h1>Organization Login</h1></body></html>"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_form.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_form.test", "template", "LOGIN"),
					resource.TestCheckResourceAttr("graviteeam_org_form.test", "enabled", "true"),
					resource.TestCheckResourceAttr("graviteeam_org_form.test", "content", "<html><body><h1>Organization Login</h1></body></html>"),
				),
			},
			{
				ResourceName:      "graviteeam_org_form.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "LOGIN",
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_form" "test" {
  template = "LOGIN"
  enabled  = false
  content  = "<html><body><h1>Welcome</h1><p>Please sign in</p></body></html>"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_form.test", "template", "LOGIN"),
					resource.TestCheckResourceAttr("graviteeam_org_form.test", "enabled", "false"),
					resource.TestCheckResourceAttr("graviteeam_org_form.test", "content", "<html><body><h1>Welcome</h1><p>Please sign in</p></body></html>"),
				),
			},
		},
	})
}
