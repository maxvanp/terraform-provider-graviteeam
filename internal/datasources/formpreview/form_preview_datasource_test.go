package formpreview_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccFormPreviewDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-form-preview"
  description = "Domain for form preview data source test"

  oidc {}
  login_settings {}
}

data "graviteeam_form_preview" "test" {
  domain_id = graviteeam_domain.test.id
  template  = "LOGIN"
  type      = "FORM"
  content   = "<html><body><h1>Hello</h1></body></html>"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_form_preview.test", "result_json"),
					resource.TestCheckResourceAttr("data.graviteeam_form_preview.test", "template", "LOGIN"),
				),
			},
		},
	})
}
