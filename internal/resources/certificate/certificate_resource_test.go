package certificate_test

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"testing"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCertificateResource_basic(t *testing.T) {
	// Generate a self-signed PKCS12 for testing (requires openssl)
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not found, skipping certificate test")
	}
	p12B64 := generateTestPKCS12(t)

	// Build the inner content JSON with the base64 cert
	innerContent, _ := json.Marshal(map[string]string{
		"content": p12B64,
		"name":    "test.p12",
	})
	innerContentStr := string(innerContent)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_domain" "test_cert" {
  name        = "test-cert-domain"
  description = "Domain for certificate test"

  oidc {}
  login_settings {}
}

resource "graviteeam_certificate" "test" {
  domain_id     = graviteeam_domain.test_cert.id
  name          = "Test Certificate"
  type          = "pkcs12-am-certificate"
  configuration = jsonencode({
    content   = %q
    storepass = "changeit"
    alias     = "mykey"
    keypass   = "changeit"
    algorithm = "RS256"
  })
}
`, innerContentStr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_certificate.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_certificate.test", "name", "Test Certificate"),
					resource.TestCheckResourceAttr("graviteeam_certificate.test", "type", "pkcs12-am-certificate"),
				),
			},
			// ImportState
			{
				ResourceName:      "graviteeam_certificate.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_certificate.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_certificate.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			// Update name
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_domain" "test_cert" {
  name        = "test-cert-domain"
  description = "Domain for certificate test"

  oidc {}
  login_settings {}
}

resource "graviteeam_certificate" "test" {
  domain_id     = graviteeam_domain.test_cert.id
  name          = "Updated Certificate"
  type          = "pkcs12-am-certificate"
  configuration = jsonencode({
    content   = %q
    storepass = "changeit"
    alias     = "mykey"
    keypass   = "changeit"
    algorithm = "RS256"
  })
}
`, innerContentStr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_certificate.test", "name", "Updated Certificate"),
					resource.TestCheckResourceAttr("graviteeam_certificate.test", "type", "pkcs12-am-certificate"),
				),
			},
		},
	})
}

func generateTestPKCS12(t *testing.T) string {
	t.Helper()

	// Generate key and cert
	keyCmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", "/tmp/test-cert-key.pem", "-out", "/tmp/test-cert.pem",
		"-days", "1", "-nodes", "-subj", "/CN=test")
	if out, err := keyCmd.CombinedOutput(); err != nil {
		t.Fatalf("generate key/cert: %v\n%s", err, out)
	}

	// Convert to PKCS12
	p12Cmd := exec.Command("openssl", "pkcs12", "-export",
		"-out", "/tmp/test-cert.p12",
		"-inkey", "/tmp/test-cert-key.pem",
		"-in", "/tmp/test-cert.pem",
		"-passout", "pass:changeit",
		"-name", "mykey")
	if out, err := p12Cmd.CombinedOutput(); err != nil {
		t.Fatalf("create pkcs12: %v\n%s", err, out)
	}

	// Read and base64 encode
	b64Cmd := exec.Command("base64", "-w0", "/tmp/test-cert.p12")
	out, err := b64Cmd.Output()
	if err != nil {
		t.Fatalf("base64 encode: %v", err)
	}

	// Cleanup
	cleanupCmd := exec.Command("rm", "-f", "/tmp/test-cert-key.pem", "/tmp/test-cert.pem", "/tmp/test-cert.p12")
	if cleanupOut, cleanupErr := cleanupCmd.CombinedOutput(); cleanupErr != nil {
		t.Fatalf("cleanup generated certificate files: %v\n%s", cleanupErr, cleanupOut)
	}

	return string(out)
}
