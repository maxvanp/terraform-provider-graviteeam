package domainmetadata_test

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccDomainMetadataDataSource_basic(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not found, skipping certificate metadata test")
	}
	p12B64 := generateTestPKCS12(t)
	innerContent, _ := json.Marshal(map[string]string{
		"content": p12B64,
		"name":    "test.p12",
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_domain" "test" {
  name        = "test-acc-domain-metadata"
  description = "Domain for domain metadata data source test"

  oidc {}
  login_settings {}
}

resource "graviteeam_password_policy" "test" {
  domain_id      = graviteeam_domain.test.id
  name           = "Metadata Password Policy"
  min_length     = 12
  default_policy = true
}

resource "graviteeam_certificate" "test" {
  domain_id     = graviteeam_domain.test.id
  name          = "Metadata Certificate"
  type          = "pkcs12-am-certificate"
  configuration = jsonencode({
    content   = %q
    storepass = "changeit"
    alias     = "mykey"
    keypass   = "changeit"
    algorithm = "RS256"
  })
}

data "graviteeam_domain_metadata" "active_password_policy" {
  domain_id = graviteeam_domain.test.id
  kind      = "active_password_policy"

  depends_on = [graviteeam_password_policy.test]
}

data "graviteeam_domain_metadata" "certificate_key" {
  domain_id      = graviteeam_domain.test.id
  kind           = "certificate_key"
  certificate_id = graviteeam_certificate.test.id
}

data "graviteeam_domain_metadata" "certificate_keys" {
  domain_id      = graviteeam_domain.test.id
  kind           = "certificate_keys"
  certificate_id = graviteeam_certificate.test.id
}
`, string(innerContent)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graviteeam_domain_metadata.active_password_policy", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_domain_metadata.certificate_key", "result_json"),
					resource.TestCheckResourceAttrSet("data.graviteeam_domain_metadata.certificate_keys", "result_json"),
				),
			},
		},
	})
}

func generateTestPKCS12(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-domain-metadata-key.pem")
	certPath := filepath.Join(tmpDir, "test-domain-metadata-cert.pem")
	p12Path := filepath.Join(tmpDir, "test-domain-metadata.p12")

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
