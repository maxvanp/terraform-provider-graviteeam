package metadata_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccMetadataDataSources_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "graviteeam_platform_metadata" "license" {
  kind = "license"
}

data "graviteeam_platform_metadata" "email_required" {
  kind = "email_required"
}

data "graviteeam_environment_metadata" "data_planes" {
  kind = "data_planes"
}

data "graviteeam_environment_metadata" "data_sources" {
  kind = "data_sources"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_platform_metadata.license", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_platform_metadata.email_required", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_environment_metadata.data_planes", "result_json"),
					resource.TestCheckResourceAttr("data.graviteeam_environment_metadata.data_sources", "result_json", "[]"),
				),
			},
		},
	})
}
