package formpreview

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	NewFormPreviewDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_form_preview"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewFormPreviewDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "template", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "content", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "result_json", false, false, true)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&FormPreviewDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	dataSource := &FormPreviewDataSource{}
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

	dataSource := &FormPreviewDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestFormatJSONHandlesEmptyJSONAndText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty": {nil, "null"},
		"text":  {[]byte("<html>preview</html>"), `"\u003chtml\u003epreview\u003c/html\u003e"`},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := formatJSON(test.raw)
			if got != test.want {
				t.Fatalf("json = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatJSONProducesStableIndentedPreviewJSON(t *testing.T) {
	t.Parallel()

	got := formatJSON([]byte(`{"content":"ok","type":"FORM"}`))
	want := "{\n  \"content\": \"ok\",\n  \"type\": \"FORM\"\n}"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFormPreviewReadPostsNormalizedPreviewRequest(t *testing.T) {
	var body map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms/preview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"content":"ok","type":"EMAIL"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &FormPreviewDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := formPreviewConfig(schemaResp.Schema, FormPreviewModel{
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Type:     types.StringValue("EMAIL"),
		Content:  types.StringValue("<html>{{user}}</html>"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if body["template"] != "login" || body["type"] != "EMAIL" || body["content"] != "<html>{{user}}</html>" {
		t.Fatalf("body = %#v", body)
	}
	var state FormPreviewModel
	if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	want := "{\n  \"content\": \"ok\",\n  \"type\": \"EMAIL\"\n}"
	if state.ResultJSON.ValueString() != want {
		t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), want)
	}
}

func TestFormPreviewReadDefaultsTypeAndReportsErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms/preview", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["type"] != "FORM" {
			t.Fatalf("type body = %#v, want FORM", body)
		}
		http.Error(w, "preview failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &FormPreviewDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := formPreviewConfig(schemaResp.Schema, FormPreviewModel{
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Type:     types.StringNull(),
		Content:  types.StringValue("<html>{{user}}</html>"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func TestFormPreviewReadReportsInvalidConfig(t *testing.T) {
	t.Parallel()

	dataSource := &FormPreviewDataSource{
		client: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id":   tftypes.Number,
				"template":    tftypes.String,
				"type":        tftypes.String,
				"content":     tftypes.String,
				"result_json": tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id":   tftypes.NewValue(tftypes.Number, 123),
				"template":    tftypes.NewValue(tftypes.String, "LOGIN"),
				"type":        tftypes.NewValue(tftypes.String, nil),
				"content":     tftypes.NewValue(tftypes.String, "<html></html>"),
				"result_json": tftypes.NewValue(tftypes.String, nil),
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

func formPreviewConfig(schema datasourceschema.Schema, model FormPreviewModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id":   tftypes.String,
				"template":    tftypes.String,
				"type":        tftypes.String,
				"content":     tftypes.String,
				"result_json": tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id":   tftypes.NewValue(tftypes.String, model.DomainID.ValueString()),
				"template":    tftypes.NewValue(tftypes.String, model.Template.ValueString()),
				"type":        formPreviewStringConfigValue(model.Type),
				"content":     tftypes.NewValue(tftypes.String, model.Content.ValueString()),
				"result_json": tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}

func formPreviewStringConfigValue(value types.String) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value.ValueString())
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
