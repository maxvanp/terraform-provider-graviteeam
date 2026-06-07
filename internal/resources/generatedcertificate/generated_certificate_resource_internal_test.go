package generatedcertificate

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGeneratedCertificateMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewGeneratedCertificateResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_generated_certificate" {
		t.Fatalf("type name = %q, want graviteeam_generated_certificate", resp.TypeName)
	}
}

func TestGeneratedCertificateSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewGeneratedCertificateResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["domain_id"]; !attr.IsRequired() {
		t.Fatal("domain_id should be required")
	}
	if attr := resp.Schema.Attributes["rotation_trigger"]; !attr.IsOptional() {
		t.Fatal("rotation_trigger should be optional")
	}
	for _, name := range []string{"id", "name", "type"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
}

func TestGeneratedCertificateConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GeneratedCertificateResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestGeneratedCertificateUpdateIsNoOp(t *testing.T) {
	t.Parallel()

	var resp resource.UpdateResponse
	NewGeneratedCertificateResource().Update(context.Background(), resource.UpdateRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected update diagnostics: %#v", resp.Diagnostics)
	}
}

func TestReadGeneratedCertificateMapsReturnedFields(t *testing.T) {
	t.Parallel()

	model := GeneratedCertificateModel{}

	readGeneratedCertificate(&model, map[string]interface{}{
		"name": "system-generated-cert",
		"type": "pem",
	})

	if got, want := model.Name.ValueString(), "system-generated-cert"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "pem"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
}

func TestReadGeneratedCertificatePreservesMissingFields(t *testing.T) {
	t.Parallel()

	model := GeneratedCertificateModel{
		Name: types.StringValue("existing-name"),
		Type: types.StringValue("existing-type"),
	}

	readGeneratedCertificate(&model, map[string]interface{}{})

	if got, want := model.Name.ValueString(), "existing-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "existing-type"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
}

func TestCertificateIDReturnsID(t *testing.T) {
	t.Parallel()

	got, ok := certificateID(map[string]interface{}{"id": "cert-id"})
	if !ok {
		t.Fatalf("expected id")
	}
	if got != "cert-id" {
		t.Fatalf("id = %q, want cert-id", got)
	}
}

func TestCertificateIDRejectsMissingEmptyOrMalformedID(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]interface{}{
		"missing":   {},
		"empty":     {"id": ""},
		"malformed": {"id": 42},
	}

	for name, result := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if id, ok := certificateID(result); ok {
				t.Fatalf("id = %q, want not ok", id)
			}
		})
	}
}
