package scope

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewScopeResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_scope"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewScopeResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "key", "name"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"description", "discovery", "expires_in", "icon_uri", "parameterized"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "discovery", "parameterized"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ScopeResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildUpdateBodyClearsRemovedScopeFields(t *testing.T) {
	t.Parallel()

	plan := ScopeModel{
		Name:          types.StringValue("updated"),
		Discovery:     types.BoolValue(false),
		Parameterized: types.BoolValue(true),
	}
	state := ScopeModel{
		Name:          types.StringValue("updated"),
		Description:   types.StringValue("old description"),
		Discovery:     types.BoolValue(true),
		ExpiresIn:     types.Int64Value(3600),
		IconURI:       types.StringValue("https://example.test/icon.png"),
		Parameterized: types.BoolValue(false),
	}

	body := (&ScopeResource{}).buildUpdateBody(plan, state)

	if body["description"] != "" {
		t.Fatalf("description = %#v, want empty string", body["description"])
	}
	if body["expiresIn"] != 0 {
		t.Fatalf("expiresIn = %#v, want 0", body["expiresIn"])
	}
	if body["iconUri"] != nil {
		t.Fatalf("iconUri = %#v, want nil", body["iconUri"])
	}
}

func TestReadIntoModelMapsScopeResponse(t *testing.T) {
	model := ScopeModel{}

	(&ScopeResource{}).readIntoModel(&model, map[string]interface{}{
		"id":            "scope-id",
		"key":           "scope-key",
		"name":          "scope-name",
		"description":   "description",
		"discovery":     false,
		"expiresIn":     float64(3600),
		"iconUri":       "https://example.test/icon.png",
		"parameterized": true,
	})

	if model.ID.ValueString() != "scope-id" {
		t.Fatalf("id = %q, want scope-id", model.ID.ValueString())
	}
	if model.Key.ValueString() != "scope-key" {
		t.Fatalf("key = %q, want scope-key", model.Key.ValueString())
	}
	if model.Name.ValueString() != "scope-name" {
		t.Fatalf("name = %q, want scope-name", model.Name.ValueString())
	}
	if model.Description.ValueString() != "description" {
		t.Fatalf("description = %q, want description", model.Description.ValueString())
	}
	if model.Discovery.ValueBool() {
		t.Fatalf("discovery = true, want false")
	}
	if model.ExpiresIn.ValueInt64() != 3600 {
		t.Fatalf("expiresIn = %d, want 3600", model.ExpiresIn.ValueInt64())
	}
	if model.IconURI.ValueString() != "https://example.test/icon.png" {
		t.Fatalf("iconUri = %q, want URL", model.IconURI.ValueString())
	}
	if !model.Parameterized.ValueBool() {
		t.Fatalf("parameterized = false, want true")
	}
}
