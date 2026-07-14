package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestPlatformMetadataMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewPlatformMetadataDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_platform_metadata"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestPlatformMetadataSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewPlatformMetadataDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "kind", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "role_id", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "result_json", false, false, true)
}

func TestPlatformMetadataConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&PlatformMetadataDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestPlatformMetadataConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	dataSource := &PlatformMetadataDataSource{}
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

func TestPlatformMetadataConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	dataSource := &PlatformMetadataDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestEnvironmentMetadataMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewEnvironmentMetadataDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_environment_metadata"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestEnvironmentMetadataSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewEnvironmentMetadataDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "kind", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "result_json", false, false, true)
}

func TestEnvironmentMetadataConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&EnvironmentMetadataDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestEnvironmentMetadataConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	dataSource := &EnvironmentMetadataDataSource{}
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

func TestEnvironmentMetadataConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	dataSource := &EnvironmentMetadataDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestPlatformMetadataRolePath(t *testing.T) {
	config := PlatformMetadataModel{
		RoleID: types.StringValue("role/id with spaces"),
	}

	path, err := platformMetadataPaths["role"](config)
	if err != nil {
		t.Fatalf("role path returned error: %v", err)
	}

	want := "platform/roles/role%2Fid%20with%20spaces"
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestPlatformMetadataRoleRequiresRoleID(t *testing.T) {
	_, err := platformMetadataPaths["role"](PlatformMetadataModel{})
	if err == nil {
		t.Fatal("expected error for missing role_id")
	}
}

func TestMetadataKindListsAreSortedAndComplete(t *testing.T) {
	t.Parallel()

	got := platformMetadataKindList()
	want := "alert_service_status, audit_event_types, email_required, flow_schema, installation, license, role, spel_grammar"
	if got != want {
		t.Fatalf("platform kinds = %q, want %q", got, want)
	}

	envKinds := metadataKindList(environmentMetadataPaths)
	if envKinds != "data_planes, data_sources" {
		t.Fatalf("environment kinds = %q", envKinds)
	}
}

func TestEnvironmentMetadataPathsAreStable(t *testing.T) {
	t.Parallel()

	for kind, path := range map[string]string{
		"data_planes":  "data-planes",
		"data_sources": "data-sources",
	} {
		if environmentMetadataPaths[kind] != path {
			t.Fatalf("%s path = %q, want %q", kind, environmentMetadataPaths[kind], path)
		}
	}
}

func TestMetadataErrorMessage(t *testing.T) {
	t.Parallel()

	err := metadataError("missing value")
	if err.Error() != "missing value" || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestPlatformMetadataReadResolvesSupportedKinds(t *testing.T) {
	var paths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/platform/installation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`{"version":"4.11.4"}`))
	})
	mux.HandleFunc("/management/platform/roles/role%2Fid%20with%20spaces", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.EscapedPath())
		_, _ = w.Write([]byte(`{"id":"role/id with spaces"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &PlatformMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []struct {
		name  string
		model PlatformMetadataModel
		want  string
	}{
		{
			name: "installation",
			model: PlatformMetadataModel{
				Kind: types.StringValue("installation"),
			},
			want: "{\n  \"version\": \"4.11.4\"\n}",
		},
		{
			name: "role",
			model: PlatformMetadataModel{
				Kind:   types.StringValue("role"),
				RoleID: types.StringValue("role/id with spaces"),
			},
			want: "{\n  \"id\": \"role/id with spaces\"\n}",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			config := platformMetadataConfig(schemaResp.Schema, tt.model)
			readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

			dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
			}
			var state PlatformMetadataModel
			if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("get state: %#v", diags)
			}
			if state.ResultJSON.ValueString() != tt.want {
				t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), tt.want)
			}
		})
	}

	wantPaths := []string{
		"/management/platform/installation",
		"/management/platform/roles/role%2Fid%20with%20spaces",
	}
	if strings.Join(paths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
}

func TestPlatformMetadataReadValidatesRequestShapeBeforeHTTP(t *testing.T) {
	dataSource := &PlatformMetadataDataSource{}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []PlatformMetadataModel{
		{Kind: types.StringValue("unknown")},
		{Kind: types.StringValue("role")},
	}

	for _, model := range cases {
		config := platformMetadataConfig(schemaResp.Schema, model)
		readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
		if !readResp.Diagnostics.HasError() {
			t.Fatalf("expected diagnostics for model %#v", model)
		}
	}
}

func TestPlatformMetadataPathBuildersResolveSupportedKinds(t *testing.T) {
	t.Parallel()

	for kind, builder := range platformMetadataPaths {
		kind, builder := kind, builder
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			path, err := builder(PlatformMetadataModel{
				RoleID: types.StringValue("role/id with spaces"),
			})
			if err != nil {
				t.Fatalf("path builder returned error: %v", err)
			}
			if path == "" {
				t.Fatal("path should not be empty")
			}
		})
	}
}

func TestPlatformMetadataReadReportsInvalidConfig(t *testing.T) {
	t.Parallel()

	dataSource := &PlatformMetadataDataSource{
		client: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"kind":        tftypes.Number,
				"role_id":     tftypes.String,
				"result_json": tftypes.String,
			}},
			map[string]tftypes.Value{
				"kind":        tftypes.NewValue(tftypes.Number, 123),
				"role_id":     tftypes.NewValue(tftypes.String, nil),
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

func TestPlatformMetadataReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/platform/installation", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "platform metadata failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &PlatformMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := platformMetadataConfig(schemaResp.Schema, PlatformMetadataModel{
		Kind: types.StringValue("installation"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func TestPlatformMetadataReadReportsFormatError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/platform/installation", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &PlatformMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := platformMetadataConfig(schemaResp.Schema, PlatformMetadataModel{
		Kind: types.StringValue("installation"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected format error diagnostics")
	}
}

func TestEnvironmentMetadataReadFormatsRemoteResult(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/data-planes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`[{"id":"default"}]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &EnvironmentMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := environmentMetadataConfig(schemaResp.Schema, EnvironmentMetadataModel{
		Kind: types.StringValue("data_planes"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var state EnvironmentMetadataModel
	if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	want := "[\n  {\n    \"id\": \"default\"\n  }\n]"
	if state.ResultJSON.ValueString() != want {
		t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), want)
	}
}

func TestEnvironmentMetadataReadValidatesKindAndReportsRemoteError(t *testing.T) {
	dataSource := &EnvironmentMetadataDataSource{}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := environmentMetadataConfig(schemaResp.Schema, EnvironmentMetadataModel{
		Kind: types.StringValue("unknown"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected unsupported kind diagnostics")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/data-sources", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "environment metadata failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	remoteDataSource := &EnvironmentMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	config = environmentMetadataConfig(schemaResp.Schema, EnvironmentMetadataModel{
		Kind: types.StringValue("data_sources"),
	})
	readResp = &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	remoteDataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func TestEnvironmentMetadataReadReportsFormatError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/data-planes", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &EnvironmentMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := environmentMetadataConfig(schemaResp.Schema, EnvironmentMetadataModel{
		Kind: types.StringValue("data_planes"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected format error diagnostics")
	}
}

func TestEnvironmentMetadataReadReportsInvalidConfig(t *testing.T) {
	t.Parallel()

	dataSource := &EnvironmentMetadataDataSource{
		client: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"kind":        tftypes.Number,
				"result_json": tftypes.String,
			}},
			map[string]tftypes.Value{
				"kind":        tftypes.NewValue(tftypes.Number, 123),
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

func platformMetadataConfig(schema datasourceschema.Schema, model PlatformMetadataModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"kind":        tftypes.String,
				"role_id":     tftypes.String,
				"result_json": tftypes.String,
			}},
			map[string]tftypes.Value{
				"kind":        tftypes.NewValue(tftypes.String, model.Kind.ValueString()),
				"role_id":     metadataStringConfigValue(model.RoleID),
				"result_json": tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}

func environmentMetadataConfig(schema datasourceschema.Schema, model EnvironmentMetadataModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"kind":        tftypes.String,
				"result_json": tftypes.String,
			}},
			map[string]tftypes.Value{
				"kind":        tftypes.NewValue(tftypes.String, model.Kind.ValueString()),
				"result_json": tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}

func metadataStringConfigValue(value types.String) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value.ValueString())
}

func assertStringAttribute(t *testing.T, attrs map[string]datasourceschema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(datasourceschema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want datasourceschema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
