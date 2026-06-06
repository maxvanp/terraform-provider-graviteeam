package protectedresource

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProtectedResourceBuildUpdateBodyClearsRemovedOptionalCollections(t *testing.T) {
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
