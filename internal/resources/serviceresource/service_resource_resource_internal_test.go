package serviceresource

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestServiceResourceMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewServiceResourceResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_service_resource" {
		t.Fatalf("type name = %q, want graviteeam_service_resource", resp.TypeName)
	}
}

func TestServiceResourceSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewServiceResourceResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

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
		t.Fatalf("id should be computed")
	}
}

func TestServiceResourceConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ServiceResourceResource{}
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
		name           string
		id             string
		wantDomainID   string
		wantResourceID string
		wantOK         bool
	}{
		{
			name:           "valid",
			id:             "domain-1/resource-1",
			wantDomainID:   "domain-1",
			wantResourceID: "resource-1",
			wantOK:         true,
		},
		{
			name:           "preserves splitN behavior",
			id:             "domain-1/resource-1/extra",
			wantDomainID:   "domain-1",
			wantResourceID: "resource-1/extra",
			wantOK:         true,
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

			gotDomainID, gotResourceID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotResourceID != tt.wantResourceID {
				t.Fatalf("resource ID = %q, want %q", gotResourceID, tt.wantResourceID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := ServiceResourceModel{
		Name:          types.StringValue("SMTP Server"),
		Type:          types.StringValue("smtp-am-resource"),
		Configuration: types.StringValue(`{"host":"smtp.example.com","password":"secret"}`),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":          "SMTP Server",
		"type":          "smtp-am-resource",
		"configuration": `{"host":"smtp.example.com","password":"secret"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := ServiceResourceModel{
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		Configuration: types.StringValue(`{"password":"real-secret"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new-name",
		"type":          "smtp-am-resource",
		"configuration": `{"password":"********"}`,
	})

	if got, want := model.Name.ValueString(), "new-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "smtp-am-resource"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"password":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}
}
