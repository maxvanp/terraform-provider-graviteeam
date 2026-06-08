package domainmetadata

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDomainMetadataPathsBuildRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		kind     string
		config   DomainMetadataModel
		wantPath string
	}{
		{
			name:     "active password policy",
			kind:     "active_password_policy",
			config:   DomainMetadataModel{},
			wantPath: "password-policies/activePolicy",
		},
		{
			name: "certificate key",
			kind: "certificate_key",
			config: DomainMetadataModel{
				CertificateID: types.StringValue("cert-1"),
			},
			wantPath: "certificates/cert-1/key",
		},
		{
			name: "certificate keys",
			kind: "certificate_keys",
			config: DomainMetadataModel{
				CertificateID: types.StringValue("cert-1"),
			},
			wantPath: "certificates/cert-1/keys",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domainMetadataPaths[tt.kind](tt.config)
			if err != nil {
				t.Fatalf("domainMetadataPaths[%q] returned error: %v", tt.kind, err)
			}
			if got != tt.wantPath {
				t.Fatalf("path = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestDomainMetadataCertificatePathsRequireCertificateID(t *testing.T) {
	t.Parallel()

	for _, kind := range []string{"certificate_key", "certificate_keys"} {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			_, err := domainMetadataPaths[kind](DomainMetadataModel{})
			if err == nil {
				t.Fatal("expected validation error")
			}
			if err != errMissingCertificateID {
				t.Fatalf("error = %#v, want errMissingCertificateID", err)
			}
		})
	}
}

func TestDomainMetadataKindListIsSorted(t *testing.T) {
	t.Parallel()

	got := domainMetadataKindList()
	want := "active_password_policy, certificate_key, certificate_keys"
	if got != want {
		t.Fatalf("kind list = %q, want %q", got, want)
	}
}

func TestFormatJSONRawString(t *testing.T) {
	t.Parallel()

	got := formatJSON([]byte("ABC123"))
	if got != `"ABC123"` {
		t.Fatalf("formatJSON() = %q, want %q", got, `"ABC123"`)
	}
}

func TestFormatJSONEmptyBody(t *testing.T) {
	t.Parallel()

	got := formatJSON(nil)
	if got != "null" {
		t.Fatalf("formatJSON() = %q, want null", got)
	}
}

func TestFormatJSONPrettyPrintsObjects(t *testing.T) {
	t.Parallel()

	got := formatJSON([]byte(`{"kid":"cert-1","use":"sig"}`))
	if !strings.Contains(got, "\n") || !strings.Contains(got, `"kid": "cert-1"`) || !strings.Contains(got, `"use": "sig"`) {
		t.Fatalf("formatJSON() = %q, want pretty JSON object", got)
	}
}
