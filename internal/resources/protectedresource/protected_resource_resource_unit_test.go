package protectedresource

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewProtectedResourceResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_protected_resource"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewProtectedResourceResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "type", "resource_identifiers"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"description", "settings_json"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "client_id", "client_secret"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
	if attr := resp.Schema.Attributes["client_secret"]; attr == nil || !attr.IsSensitive() {
		t.Fatalf("client_secret should be sensitive")
	}
	if _, ok := resp.Schema.Blocks["feature"]; !ok {
		t.Fatalf("missing feature block")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ProtectedResourceResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestProtectedResourceBuildCreateBodyIncludesTypeAndSettings(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	plan := ProtectedResourceModel{
		Name:                types.StringValue("mcp"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.com/mcp")},
		SettingsJSON:        types.StringValue(`{"authorizationServer":"as-1","enabled":true}`),
		Features: []ProtectedResourceFeatureModel{
			{
				Key:         types.StringValue("list_items"),
				Type:        types.StringValue("MCP_TOOL"),
				Description: types.StringValue("List items"),
				Scopes:      []types.String{types.StringValue("openid"), types.StringValue("profile")},
			},
		},
	}

	body, err := resource.buildCreateBody(plan)
	if err != nil {
		t.Fatal(err)
	}

	if body["type"] != "MCP_SERVER" {
		t.Fatalf("type = %#v, want MCP_SERVER", body["type"])
	}

	settings, ok := body["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("settings = %#v, want object", body["settings"])
	}
	if settings["authorizationServer"] != "as-1" || settings["enabled"] != true {
		t.Fatalf("settings = %#v", settings)
	}

	features, ok := body["features"].([]map[string]interface{})
	if !ok || len(features) != 1 {
		t.Fatalf("features = %#v, want one feature", body["features"])
	}
	wantScopes := []string{"openid", "profile"}
	if !reflect.DeepEqual(features[0]["scopes"], wantScopes) {
		t.Fatalf("feature scopes = %#v, want %#v", features[0]["scopes"], wantScopes)
	}
}

func TestProtectedResourceBuildUpdateBodyClearsRemovedOptionalCollections(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	plan := ProtectedResourceModel{
		Name:                types.StringValue("updated"),
		Description:         types.StringValue("updated description"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.com/mcp")},
		SettingsJSON:        types.StringNull(),
		Features:            nil,
	}
	state := &ProtectedResourceModel{
		SettingsJSON: types.StringValue(`{"existing":true}`),
		Features: []ProtectedResourceFeatureModel{
			{
				Key:  types.StringValue("existing"),
				Type: types.StringValue("MCP_TOOL"),
			},
		},
	}

	body, err := resource.buildUpdateBody(plan, state)
	if err != nil {
		t.Fatal(err)
	}

	features, ok := body["features"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected removed feature block to be sent as an empty feature list, got %#v", body["features"])
	}
	if len(features) != 0 {
		t.Fatalf("expected empty feature list, got %#v", features)
	}

	settings, ok := body["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected removed settings_json to be sent as an empty settings object, got %#v", body["settings"])
	}
	if len(settings) != 0 {
		t.Fatalf("expected empty settings object, got %#v", settings)
	}
}

func TestProtectedResourceBuildUpdateBodyRejectsInvalidSettingsJSON(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	plan := ProtectedResourceModel{
		Name:                types.StringValue("mcp"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.com/mcp")},
		SettingsJSON:        types.StringValue(`{"broken":`),
	}

	_, err := resource.buildUpdateBody(plan, nil)
	if err == nil {
		t.Fatal("expected invalid settings_json error")
	}
	if !strings.Contains(err.Error(), "settings_json must be a valid JSON object") {
		t.Fatalf("error = %q", err)
	}
}

func TestProtectedResourceReadIntoModelMapsAPIFields(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	model := ProtectedResourceModel{}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":                  "resource-1",
		"name":                "mcp",
		"description":         "MCP server",
		"type":                "mcp_server",
		"clientId":            "client-1",
		"resourceIdentifiers": []interface{}{"https://api.example.com/mcp", 42, "https://api.example.com/secondary"},
		"features": []interface{}{
			map[string]interface{}{
				"key":         "list_items",
				"type":        "mcp_tool",
				"description": "List items",
				"scopes":      []interface{}{"openid", true, "profile"},
			},
			"ignored",
		},
	})

	if model.ID.ValueString() != "resource-1" ||
		model.Name.ValueString() != "mcp" ||
		model.Description.ValueString() != "MCP server" ||
		model.Type.ValueString() != "MCP_SERVER" ||
		model.ClientID.ValueString() != "client-1" {
		t.Fatalf("model = %#v", model)
	}

	wantIdentifiers := []types.String{types.StringValue("https://api.example.com/mcp"), types.StringValue("https://api.example.com/secondary")}
	if !reflect.DeepEqual(model.ResourceIdentifiers, wantIdentifiers) {
		t.Fatalf("resource identifiers = %#v, want %#v", model.ResourceIdentifiers, wantIdentifiers)
	}

	if len(model.Features) != 1 {
		t.Fatalf("features = %#v, want one valid feature", model.Features)
	}
	feature := model.Features[0]
	if feature.Key.ValueString() != "list_items" ||
		feature.Type.ValueString() != "MCP_TOOL" ||
		feature.Description.ValueString() != "List items" {
		t.Fatalf("feature = %#v", feature)
	}
	wantScopes := []types.String{types.StringValue("openid"), types.StringValue("profile")}
	if !reflect.DeepEqual(feature.Scopes, wantScopes) {
		t.Fatalf("feature scopes = %#v, want %#v", feature.Scopes, wantScopes)
	}
}

func TestProtectedResourceReadSecretIntoModelMapsCreationSecrets(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	model := ProtectedResourceModel{}

	resource.readSecretIntoModel(&model, map[string]interface{}{
		"id":           "resource-1",
		"clientId":     "client-1",
		"clientSecret": "secret-1",
	})

	if model.ID.ValueString() != "resource-1" ||
		model.ClientID.ValueString() != "client-1" ||
		model.ClientSecret.ValueString() != "secret-1" {
		t.Fatalf("model = %#v", model)
	}
}
