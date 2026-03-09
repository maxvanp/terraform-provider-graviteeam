package serviceresource_test

import (
	"fmt"
	"testing"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccServiceResourceResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_svcres" {
  name        = "test-svcres-domain"
  description = "Domain for service resource test"

  oidc {}
  login_settings {}
}

resource "graviteeam_service_resource" "test" {
  domain_id     = graviteeam_domain.test_svcres.id
  name          = "Test SMTP Resource"
  type          = "smtp-am-resource"
  configuration = jsonencode({
    host           = "localhost"
    port           = 1025
    from           = "noreply@test.local"
    protocol       = "smtp"
    authentication = false
    startTls       = false
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_service_resource.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_service_resource.test", "name", "Test SMTP Resource"),
					resource.TestCheckResourceAttr("graviteeam_service_resource.test", "type", "smtp-am-resource"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_service_resource.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_service_resource.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_service_resource.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			// Update name
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test_svcres" {
  name        = "test-svcres-domain"
  description = "Domain for service resource test"

  oidc {}
  login_settings {}
}

resource "graviteeam_service_resource" "test" {
  domain_id     = graviteeam_domain.test_svcres.id
  name          = "Updated SMTP Resource"
  type          = "smtp-am-resource"
  configuration = jsonencode({
    host           = "localhost"
    port           = 1025
    from           = "noreply@test.local"
    protocol       = "smtp"
    authentication = false
    startTls       = false
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_service_resource.test", "name", "Updated SMTP Resource"),
				),
			},
		},
	})
}
