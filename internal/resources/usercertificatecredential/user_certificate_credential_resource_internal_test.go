package usercertificatecredential

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := UserCertificateCredentialModel{
		CertificatePEM: types.StringValue("-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n"),
	}

	got := buildCreateBody(plan)
	want := map[string]interface{}{
		"certificatePem": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelMapsCertificateCredential(t *testing.T) {
	t.Parallel()

	model := UserCertificateCredentialModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":                      "credential-id",
		"certificatePem":          "pem",
		"certificateThumbprint":   "thumbprint",
		"certificateSubjectDN":    "CN=subject",
		"certificateSerialNumber": "serial",
		"certificateIssuerDN":     "CN=issuer",
		"certificateExpiresAt":    "2026-06-07T12:00:00Z",
		"username":                "alice",
	})

	assertStringAttr(t, model.ID, "id", "credential-id")
	assertStringAttr(t, model.CertificatePEM, "certificate pem", "pem")
	assertStringAttr(t, model.CertificateThumbprint, "certificate thumbprint", "thumbprint")
	assertStringAttr(t, model.CertificateSubjectDN, "certificate subject dn", "CN=subject")
	assertStringAttr(t, model.CertificateSerialNumber, "certificate serial number", "serial")
	assertStringAttr(t, model.CertificateIssuerDN, "certificate issuer dn", "CN=issuer")
	assertStringAttr(t, model.CertificateExpiresAt, "certificate expires at", "2026-06-07T12:00:00Z")
	assertStringAttr(t, model.Username, "username", "alice")
}

func TestReadIntoModelNullsMissingComputedFields(t *testing.T) {
	t.Parallel()

	model := UserCertificateCredentialModel{
		CertificateThumbprint:   types.StringValue("thumbprint"),
		CertificateSubjectDN:    types.StringValue("CN=subject"),
		CertificateSerialNumber: types.StringValue("serial"),
		CertificateIssuerDN:     types.StringValue("CN=issuer"),
		CertificateExpiresAt:    types.StringValue("2026-06-07T12:00:00Z"),
		Username:                types.StringValue("alice"),
	}

	readIntoModel(&model, map[string]interface{}{})

	assertNullString(t, model.CertificateThumbprint, "certificate thumbprint")
	assertNullString(t, model.CertificateSubjectDN, "certificate subject dn")
	assertNullString(t, model.CertificateSerialNumber, "certificate serial number")
	assertNullString(t, model.CertificateIssuerDN, "certificate issuer dn")
	assertNullString(t, model.CertificateExpiresAt, "certificate expires at")
	assertNullString(t, model.Username, "username")
}

func TestTimestampString(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input interface{}
		want  string
		null  bool
	}{
		"string": {
			input: "2026-06-07T12:00:00Z",
			want:  "2026-06-07T12:00:00Z",
		},
		"float": {
			input: float64(1780833600000),
			want:  "1780833600000",
		},
		"nil": {
			input: nil,
			null:  true,
		},
		"malformed": {
			input: map[string]interface{}{},
			null:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := timestampString(tc.input)
			if tc.null {
				assertNullString(t, got, name)
				return
			}
			assertStringAttr(t, got, name, tc.want)
		})
	}
}

func assertStringAttr(t *testing.T, got types.String, name, want string) {
	t.Helper()

	if got.IsNull() || got.IsUnknown() {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
	if got.ValueString() != want {
		t.Fatalf("%s = %q, want %q", name, got.ValueString(), want)
	}
}

func assertNullString(t *testing.T, got types.String, name string) {
	t.Helper()

	if !got.IsNull() {
		t.Fatalf("%s should be null, got %q", name, got.ValueString())
	}
}
