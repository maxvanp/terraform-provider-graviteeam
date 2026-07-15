package plugins_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccPluginsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "graviteeam_plugins" "identities" {
  category = "identities"
}

data "graviteeam_plugins" "identity_http" {
  category  = "identities"
  plugin_id = "http-am-idp"
}

data "graviteeam_plugins" "identity_http_schema" {
  category  = "identities"
  plugin_id = "http-am-idp"
  schema    = true
}

data "graviteeam_plugins" "policy_documentation" {
  category      = "policies"
  plugin_id      = "groovy"
  documentation = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_plugins.identities", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_plugins.identity_http", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_plugins.identity_http_schema", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_plugins.policy_documentation", "result_json"),
				),
			},
		},
	})
}
