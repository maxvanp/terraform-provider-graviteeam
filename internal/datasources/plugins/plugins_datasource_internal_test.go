package plugins

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
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

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	dataSource := &PluginsDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	dataSource := &PluginsDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
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

			got := formatJSONResult(test.raw)
			if got != test.want {
				t.Fatalf("result = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatJSONResultProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got := formatJSONResult([]byte(`{"id":"ldap","enabled":true}`))
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

	got := formatJSONResult([]byte("line\nbreak"))
	var decoded string
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("formatted text is not a JSON string: %v", err)
	}
	if decoded != "line\nbreak" {
		t.Fatalf("decoded = %q, want original text", decoded)
	}
}

func TestPluginsReadCatalogSchemaAndDocumentation(t *testing.T) {
	var paths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/platform/plugins/factors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`[{"id":"otp"}]`))
	})
	mux.HandleFunc("/management/platform/plugins/factors/otp/schema", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`{"id":"schema"}`))
	})
	mux.HandleFunc("/management/platform/plugins/policies/groovy/documentation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`# docs`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &PluginsDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []struct {
		name  string
		model PluginsModel
		want  string
	}{
		{
			name: "catalog",
			model: PluginsModel{
				Category: types.StringValue("factors"),
			},
			want: "[\n  {\n    \"id\": \"otp\"\n  }\n]",
		},
		{
			name: "schema",
			model: PluginsModel{
				Category: types.StringValue("factors"),
				PluginID: types.StringValue("otp"),
				Schema:   types.BoolValue(true),
			},
			want: "{\n  \"id\": \"schema\"\n}",
		},
		{
			name: "documentation",
			model: PluginsModel{
				Category:      types.StringValue("policies"),
				PluginID:      types.StringValue("groovy"),
				Documentation: types.BoolValue(true),
			},
			want: "\"# docs\"",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			config := pluginsConfig(t, schemaResp.Schema, tt.model)
			readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

			dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
			}
			var state PluginsModel
			if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("get state: %#v", diags)
			}
			if state.ResultJSON.ValueString() != tt.want {
				t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), tt.want)
			}
		})
	}

	wantPaths := []string{
		"/management/platform/plugins/factors",
		"/management/platform/plugins/factors/otp/schema",
		"/management/platform/plugins/policies/groovy/documentation",
	}
	if strings.Join(paths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
}

func TestPluginsReadValidatesRequestShapeBeforeHTTP(t *testing.T) {
	dataSource := &PluginsDataSource{}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []PluginsModel{
		{Category: types.StringValue("unknown")},
		{Category: types.StringValue("factors"), Schema: types.BoolValue(true)},
		{Category: types.StringValue("policies"), Documentation: types.BoolValue(true)},
		{Category: types.StringValue("factors"), PluginID: types.StringValue("otp"), Schema: types.BoolValue(true), Documentation: types.BoolValue(true)},
		{Category: types.StringValue("factors"), PluginID: types.StringValue("otp"), Documentation: types.BoolValue(true)},
	}

	for _, model := range cases {
		config := pluginsConfig(t, schemaResp.Schema, model)
		readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
		if !readResp.Diagnostics.HasError() {
			t.Fatalf("expected diagnostics for model %#v", model)
		}
	}
}

func TestPluginsReadReportsInvalidConfig(t *testing.T) {
	t.Parallel()

	dataSource := &PluginsDataSource{
		client: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"category":      tftypes.Number,
				"plugin_id":     tftypes.String,
				"schema":        tftypes.Bool,
				"documentation": tftypes.Bool,
				"result_json":   tftypes.String,
			}},
			map[string]tftypes.Value{
				"category":      tftypes.NewValue(tftypes.Number, 123),
				"plugin_id":     tftypes.NewValue(tftypes.String, nil),
				"schema":        tftypes.NewValue(tftypes.Bool, nil),
				"documentation": tftypes.NewValue(tftypes.Bool, nil),
				"result_json":   tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schemaResp.Schema,
	}

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected invalid config diagnostics")
	}
}

func TestPluginsReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/platform/plugins/factors", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "plugins failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &PluginsDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := pluginsConfig(t, schemaResp.Schema, PluginsModel{
		Category: types.StringValue("factors"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func pluginsConfig(t *testing.T, schema datasourceschema.Schema, model PluginsModel) tfsdk.Config {
	t.Helper()

	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"category":      tftypes.String,
			"plugin_id":     tftypes.String,
			"schema":        tftypes.Bool,
			"documentation": tftypes.Bool,
			"result_json":   tftypes.String,
		}},
		map[string]tftypes.Value{
			"category":      tftypes.NewValue(tftypes.String, model.Category.ValueString()),
			"plugin_id":     pluginStringConfigValue(model.PluginID),
			"schema":        pluginBoolConfigValue(model.Schema),
			"documentation": pluginBoolConfigValue(model.Documentation),
			"result_json":   tftypes.NewValue(tftypes.String, nil),
		},
	)

	return tfsdk.Config{
		Raw:    raw,
		Schema: schema,
	}
}

func pluginStringConfigValue(value types.String) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value.ValueString())
}

func pluginBoolConfigValue(value types.Bool) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.Bool, nil)
	}
	return tftypes.NewValue(tftypes.Bool, value.ValueBool())
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
