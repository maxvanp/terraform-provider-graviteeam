package domaincertificatesettings_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccDomainCertificateSettingsResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set, skipping acceptance test")
	}
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not found, skipping domain certificate settings test")
	}
	p12B64 := generateTestPKCS12(t)
	innerContent, _ := json.Marshal(map[string]string{
		"content": p12B64,
		"name":    "test.p12",
	})
	innerContentStr := string(innerContent)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_domain" "test" {
  name        = "test-acc-domain-cert-settings"
  description = "Domain for certificate settings test"

  oidc {}
  login_settings {}
}

resource "graviteeam_certificate" "test" {
  domain_id     = graviteeam_domain.test.id
  name          = "Fallback Certificate"
  type          = "pkcs12-am-certificate"
  configuration = jsonencode({
    content   = %q
    storepass = "changeit"
    alias     = "mykey"
    keypass   = "changeit"
    algorithm = "RS256"
  })
}

resource "graviteeam_domain_certificate_settings" "test" {
  domain_id               = graviteeam_domain.test.id
  fallback_certificate_id = graviteeam_certificate.test.id
}
`, innerContentStr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("graviteeam_domain_certificate_settings.test", "id", "graviteeam_domain.test", "id"),
					resource.TestCheckResourceAttrPair("graviteeam_domain_certificate_settings.test", "domain_id", "graviteeam_domain.test", "id"),
					resource.TestCheckResourceAttrPair("graviteeam_domain_certificate_settings.test", "fallback_certificate_id", "graviteeam_certificate.test", "id"),
				),
			},
			{
				ResourceName:      "graviteeam_domain_certificate_settings.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_domain_certificate_settings.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_domain_certificate_settings.test")
					}
					return rs.Primary.Attributes["domain_id"], nil
				},
			},
		},
	})
}

func generateTestPKCS12(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "test-cert-settings-key.pem")
	certPath := filepath.Join(dir, "test-cert-settings.pem")
	p12Path := filepath.Join(dir, "test-cert-settings.p12")

	keyCmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyPath, "-out", certPath,
		"-days", "1", "-nodes", "-subj", "/CN=test")
	if out, err := keyCmd.CombinedOutput(); err != nil {
		t.Fatalf("generate key/cert: %v\n%s", err, out)
	}

	p12Cmd := exec.Command("openssl", "pkcs12", "-export",
		"-out", p12Path,
		"-inkey", keyPath,
		"-in", certPath,
		"-passout", "pass:changeit",
		"-name", "mykey")
	if out, err := p12Cmd.CombinedOutput(); err != nil {
		t.Fatalf("create pkcs12: %v\n%s", err, out)
	}

	b64Cmd := exec.Command("base64", "-w0", p12Path)
	out, err := b64Cmd.Output()
	if err != nil {
		t.Fatalf("base64 encode: %v", err)
	}

	return string(out)
}
