package domaincertificatesettings

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDomainCertificateSettingsMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewDomainCertificateSettingsResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_domain_certificate_settings" {
		t.Fatalf("type name = %q, want graviteeam_domain_certificate_settings", resp.TypeName)
	}
}

func TestDomainCertificateSettingsSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewDomainCertificateSettingsResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "fallback_certificate_id"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; !attr.IsComputed() {
		t.Fatal("id should be computed")
	}
}

func TestDomainCertificateSettingsConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &DomainCertificateSettingsResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildBodySetsFallbackCertificate(t *testing.T) {
	t.Parallel()

	got := buildBody("cert-1")
	want := map[string]interface{}{"fallbackCertificate": "cert-1"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildDeleteBodyClearsFallbackCertificate(t *testing.T) {
	t.Parallel()

	got := buildDeleteBody()

	if value, ok := got["fallbackCertificate"]; !ok || value != nil {
		t.Fatalf("fallbackCertificate = %#v, want nil", got)
	}
}

func TestReadFallbackCertificateID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		domain map[string]interface{}
		want   string
	}{
		"present": {
			domain: map[string]interface{}{
				"certificateSettings": map[string]interface{}{
					"fallbackCertificate": "cert-1",
				},
			},
			want: "cert-1",
		},
		"missing settings": {
			domain: map[string]interface{}{},
			want:   "",
		},
		"wrong settings type": {
			domain: map[string]interface{}{"certificateSettings": "invalid"},
			want:   "",
		},
		"missing fallback": {
			domain: map[string]interface{}{"certificateSettings": map[string]interface{}{}},
			want:   "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := readFallbackCertificateID(test.domain); got != test.want {
				t.Fatalf("fallback = %q, want %q", got, test.want)
			}
		})
	}
}
