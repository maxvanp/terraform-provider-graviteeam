package generatedcertificate_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccGeneratedCertificateResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-generated-cert"
  description = "Domain for generated certificate test"

  oidc {}
  login_settings {}
}

resource "graviteeam_generated_certificate" "test" {
  domain_id        = graviteeam_domain.test.id
  rotation_trigger = "initial"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_generated_certificate.test", "id"),
					resource.TestCheckResourceAttrPair("graviteeam_generated_certificate.test", "domain_id", "graviteeam_domain.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_generated_certificate.test", "rotation_trigger", "initial"),
					resource.TestCheckResourceAttrSet("graviteeam_generated_certificate.test", "name"),
					resource.TestCheckResourceAttrSet("graviteeam_generated_certificate.test", "type"),
				),
			},
			{
				ResourceName:      "graviteeam_generated_certificate.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_generated_certificate.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_generated_certificate.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"rotation_trigger"},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_domain" "test" {
  name        = "test-acc-generated-cert"
  description = "Domain for generated certificate test"

  oidc {}
  login_settings {}
}

resource "graviteeam_generated_certificate" "test" {
  domain_id        = graviteeam_domain.test.id
  rotation_trigger = "replacement"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_generated_certificate.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_generated_certificate.test", "rotation_trigger", "replacement"),
					resource.TestCheckResourceAttrSet("graviteeam_generated_certificate.test", "name"),
					resource.TestCheckResourceAttrSet("graviteeam_generated_certificate.test", "type"),
				),
			},
		},
	})
}
