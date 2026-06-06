package usercertificatecredential_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccUserCertificateCredentialResource_basic(t *testing.T) {
	certificatePEM := strings.TrimSuffix(testCertificatePEM(t, "test-acc-user-cert-credential"), "\n")
	expectedCertificatePEM := certificatePEM + "\n"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "graviteeam_domain" "test" {
  name        = "test-acc-user-cert-credential"
  description = "Acceptance test domain"

  oidc {}
  login_settings {}
}

resource "graviteeam_user" "test" {
  domain_id        = graviteeam_domain.test.id
  username         = "test-acc-user-cert-credential"
  email            = "test-acc-user-cert-credential@example.com"
  first_name       = "Certificate"
  last_name        = "Credential"
  pre_registration = true
}

resource "graviteeam_user_certificate_credential" "test" {
  domain_id       = graviteeam_domain.test.id
  user_id         = graviteeam_user.test.id
  certificate_pem = <<EOT
%s
EOT
}
`, certificatePEM),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_user_certificate_credential.test", "id"),
					resource.TestCheckResourceAttrPair("graviteeam_user_certificate_credential.test", "domain_id", "graviteeam_domain.test", "id"),
					resource.TestCheckResourceAttrPair("graviteeam_user_certificate_credential.test", "user_id", "graviteeam_user.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_user_certificate_credential.test", "certificate_pem", expectedCertificatePEM),
					resource.TestCheckResourceAttrSet("graviteeam_user_certificate_credential.test", "certificate_thumbprint"),
					resource.TestCheckResourceAttr("graviteeam_user_certificate_credential.test", "certificate_subject_dn", "CN=test-acc-user-cert-credential"),
					resource.TestCheckResourceAttrSet("graviteeam_user_certificate_credential.test", "certificate_serial_number"),
					resource.TestCheckResourceAttr("graviteeam_user_certificate_credential.test", "certificate_issuer_dn", "CN=test-acc-user-cert-credential"),
					resource.TestCheckResourceAttrSet("graviteeam_user_certificate_credential.test", "certificate_expires_at"),
					resource.TestCheckResourceAttr("graviteeam_user_certificate_credential.test", "username", "test-acc-user-cert-credential"),
				),
			},
			{
				ResourceName:      "graviteeam_user_certificate_credential.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graviteeam_user_certificate_credential.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: graviteeam_user_certificate_credential.test")
					}
					return rs.Primary.Attributes["domain_id"] + "/" + rs.Primary.Attributes["user_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
		},
	})
}

func testCertificatePEM(t *testing.T, commonName string) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generate serial number: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
}
