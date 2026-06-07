package plugins

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewPluginsDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_plugins"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewPluginsDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "category", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "plugin_id", false, true, false)
	assertBoolAttribute(t, resp.Schema.Attributes, "schema", false, true, false)
	assertBoolAttribute(t, resp.Schema.Attributes, "documentation", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "result_json", false, false, true)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&PluginsDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestFormatJSONResultHandlesEmptyJSONAndText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty": {nil, "null"},
		"text":  {[]byte("plain text"), `"plain text"`},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := formatJSONResult(test.raw)
			if err != nil {
				t.Fatalf("format result: %v", err)
			}
			if got != test.want {
				t.Fatalf("result = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatJSONResultProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatJSONResult([]byte(`{"id":"ldap","enabled":true}`))
	if err != nil {
		t.Fatalf("format result: %v", err)
	}
	want := "{\n  \"enabled\": true,\n  \"id\": \"ldap\"\n}"
	if got != want {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPluginCategoryListContainsEveryAllowedCategory(t *testing.T) {
	t.Parallel()

	got := pluginCategoryList()
	for category := range allowedPluginCategories {
		if !strings.Contains(got, category) {
			t.Fatalf("category list %q does not contain %q", got, category)
		}
	}
}

func TestFormatJSONResultEscapesInvalidJSONText(t *testing.T) {
	t.Parallel()

	got, err := formatJSONResult([]byte("line\nbreak"))
	if err != nil {
		t.Fatalf("format result: %v", err)
	}
	var decoded string
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("formatted text is not a JSON string: %v", err)
	}
	if decoded != "line\nbreak" {
		t.Fatalf("decoded = %q, want original text", decoded)
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

func assertBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.BoolAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
