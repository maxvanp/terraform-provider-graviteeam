package permissionsmetadata_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccPermissionsMetadataDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-permissions-metadata"
  description = "Domain for permissions metadata data source test"

  oidc {}
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id = graviteeam_domain.test.id
  name      = "Permissions Metadata Application"
  type      = "WEB"

  oauth_settings {
    redirect_uris  = ["https://example.com/auth"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
    scopes         = ["openid"]
  }
}

resource "graviteeam_protected_resource" "test" {
  domain_id            = graviteeam_domain.test.id
  name                 = "Permissions Metadata Protected Resource"
  type                 = "MCP_SERVER"
  resource_identifiers = ["https://api.example.com/permissions-metadata/mcp"]

  feature {
    key         = "list_items"
    type        = "MCP_TOOL"
    description = "List items"
    scopes      = ["openid"]
  }
}

data "graviteeam_permissions_metadata" "environment" {
  kind = "environment_member_permissions"
}

data "graviteeam_permissions_metadata" "domain" {
  kind      = "domain_member_permissions"
  domain_id = graviteeam_domain.test.id
}

data "graviteeam_permissions_metadata" "application" {
  kind           = "application_member_permissions"
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
}

data "graviteeam_permissions_metadata" "protected_resource" {
  kind                  = "protected_resource_member_permissions"
  domain_id             = graviteeam_domain.test.id
  protected_resource_id = graviteeam_protected_resource.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_permissions_metadata.environment", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_permissions_metadata.domain", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_permissions_metadata.application", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_permissions_metadata.protected_resource", "result_json"),
				),
			},
		},
	})
}
