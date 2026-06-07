package authorizationengine

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewAuthorizationEngineResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_authorization_engine"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewAuthorizationEngineResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "configuration", true, false, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&AuthorizationEngineResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		id           string
		wantDomainID string
		wantEngineID string
		wantOK       bool
	}{
		{
			name:         "valid",
			id:           "domain-1/engine-1",
			wantDomainID: "domain-1",
			wantEngineID: "engine-1",
			wantOK:       true,
		},
		{
			name:         "preserves splitN behavior",
			id:           "domain-1/engine-1/extra",
			wantDomainID: "domain-1",
			wantEngineID: "engine-1/extra",
			wantOK:       true,
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

			gotDomainID, gotEngineID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotEngineID != tt.wantEngineID {
				t.Fatalf("engine ID = %q, want %q", gotEngineID, tt.wantEngineID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := AuthorizationEngineModel{
		Name:          types.StringValue("OpenFGA"),
		Type:          types.StringValue("openfga"),
		Configuration: types.StringValue(`{"storeId":"store-1"}`),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":          "OpenFGA",
		"type":          "openfga",
		"configuration": `{"storeId":"store-1"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	model := AuthorizationEngineModel{
		ID:            types.StringValue("old-id"),
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		Configuration: types.StringValue(`{"storeId":"old"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"id":            "engine-1",
		"name":          "OpenFGA",
		"type":          "openfga",
		"configuration": `{"storeId":"new"}`,
	})

	if got, want := model.ID.ValueString(), "engine-1"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.Name.ValueString(), "OpenFGA"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "openfga"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"storeId":"new"}`; got != want {
		t.Fatalf("configuration = %q, want %q", got, want)
	}
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
