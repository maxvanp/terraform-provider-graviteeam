package applicationmetadata_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccApplicationMetadataDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-application-metadata"
  description = "Domain for application metadata data source test"

  oidc {
    allow_localhost_redirect_uri   = true
    allow_http_scheme_redirect_uri = true
  }
  login_settings {}
}

resource "graviteeam_application" "test" {
  domain_id   = graviteeam_domain.test.id
  name        = "Application Metadata Test"
  type        = "WEB"
  description = "Application metadata acceptance test"

  oauth_settings {
    redirect_uris  = ["https://example.com/auth"]
    grant_types    = ["authorization_code"]
    response_types = ["code"]
    scopes         = ["openid"]
  }
}

data "graviteeam_application_metadata" "analytics" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  kind           = "analytics"
  type           = "GROUP_BY"
  field          = "application"
  interval       = 86400000
}

data "graviteeam_application_metadata" "resources" {
  domain_id      = graviteeam_domain.test.id
  application_id = graviteeam_application.test.id
  kind           = "resources"
  size           = 10
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_application_metadata.analytics", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_application_metadata.resources", "result_json"),
					resource.TestCheckResourceAttr("data.graviteeam_application_metadata.analytics", "kind", "analytics"),
					resource.TestCheckResourceAttr("data.graviteeam_application_metadata.resources", "kind", "resources"),
				),
			},
		},
	})
}
