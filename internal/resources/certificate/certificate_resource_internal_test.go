package certificate

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCertificateMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewCertificateResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_certificate" {
		t.Fatalf("type name = %q, want graviteeam_certificate", resp.TypeName)
	}
}

func TestCertificateSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewCertificateResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "type", "configuration"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["configuration"]; !attr.IsSensitive() {
		t.Fatal("configuration should be sensitive")
	}
	if attr := resp.Schema.Attributes["id"]; !attr.IsComputed() {
		t.Fatal("id should be computed")
	}
}

func TestCertificateConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &CertificateResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		id                string
		wantDomainID      string
		wantCertificateID string
		wantOK            bool
	}{
		{
			name:              "valid",
			id:                "domain-1/cert-1",
			wantDomainID:      "domain-1",
			wantCertificateID: "cert-1",
			wantOK:            true,
		},
		{
			name:              "preserves splitN behavior",
			id:                "domain-1/cert-1/extra",
			wantDomainID:      "domain-1",
			wantCertificateID: "cert-1/extra",
			wantOK:            true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotCertificateID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotCertificateID != tt.wantCertificateID {
				t.Fatalf("certificate ID = %q, want %q", gotCertificateID, tt.wantCertificateID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := CertificateModel{
		Name:          types.StringValue("JWT Signing Certificate"),
		Type:          types.StringValue("pkcs12-am-certificate"),
		Configuration: types.StringValue(`{"storepass":"changeit","keypass":"changeit","algorithm":"RS256"}`),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":          "JWT Signing Certificate",
		"type":          "pkcs12-am-certificate",
		"configuration": `{"storepass":"changeit","keypass":"changeit","algorithm":"RS256"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := CertificateModel{
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		Configuration: types.StringValue(`{"storepass":"real-secret","keypass":"real-secret"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new-name",
		"type":          "pkcs12-am-certificate",
		"configuration": `{"storepass":"********","keypass":"********"}`,
	})

	if got, want := model.Name.ValueString(), "new-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "pkcs12-am-certificate"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"storepass":"real-secret","keypass":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}
}
