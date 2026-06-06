package selfmetadata_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccSelfMetadataDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "graviteeam_self_metadata" "current_user" {
  kind = "current_user"
}

data "graviteeam_self_metadata" "newsletter_taglines" {
  kind = "newsletter_taglines"
}

data "graviteeam_self_metadata" "notifications" {
  kind = "notifications"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_self_metadata.current_user", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_self_metadata.newsletter_taglines", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_self_metadata.notifications", "result_json"),
				),
			},
		},
	})
}
