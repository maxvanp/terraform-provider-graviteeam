package permissionsmetadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestPermissionsMetadataNewMetadataAndConfigure(t *testing.T) {
	t.Parallel()

	dataSource, ok := NewPermissionsMetadataDataSource().(*PermissionsMetadataDataSource)
	if !ok {
		t.Fatalf("data source type = %T, want *PermissionsMetadataDataSource", NewPermissionsMetadataDataSource())
	}

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &metadataResp)
	if got, want := metadataResp.TypeName, "graviteeam_permissions_metadata"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}

	var configureResp datasource.ConfigureResponse
	dataSource.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", configureResp.Diagnostics)
	}
	if dataSource.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestPermissionsMetadataConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&PermissionsMetadataDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestPermissionsMetadataConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	dataSource := &PermissionsMetadataDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestPermissionsMetadataReadResolvesSupportedKinds(t *testing.T) {
	var paths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/members/permissions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`["ENVIRONMENT[READ]"]`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/members/permissions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`["DOMAIN[READ]"]`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/applications/app-1/members/permissions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`["APPLICATION[READ]"]`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/protected-resources/resource-1/members/permissions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`["PROTECTED_RESOURCE[READ]"]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &PermissionsMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []struct {
		name  string
		model PermissionsMetadataModel
		want  string
	}{
		{
			name: "environment member permissions",
			model: PermissionsMetadataModel{
				Kind: types.StringValue("environment_member_permissions"),
			},
			want: "[\n  \"ENVIRONMENT[READ]\"\n]",
		},
		{
			name: "domain member permissions",
			model: PermissionsMetadataModel{
				Kind:     types.StringValue("domain_member_permissions"),
				DomainID: types.StringValue("domain-1"),
			},
			want: "[\n  \"DOMAIN[READ]\"\n]",
		},
		{
			name: "application member permissions",
			model: PermissionsMetadataModel{
				Kind:          types.StringValue("application_member_permissions"),
				DomainID:      types.StringValue("domain-1"),
				ApplicationID: types.StringValue("app-1"),
			},
			want: "[\n  \"APPLICATION[READ]\"\n]",
		},
		{
			name: "protected resource member permissions",
			model: PermissionsMetadataModel{
				Kind:                types.StringValue("protected_resource_member_permissions"),
				DomainID:            types.StringValue("domain-1"),
				ProtectedResourceID: types.StringValue("resource-1"),
			},
			want: "[\n  \"PROTECTED_RESOURCE[READ]\"\n]",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			config := permissionsMetadataConfig(schemaResp.Schema, tt.model)
			readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

			dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
			}
			var state PermissionsMetadataModel
			if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("get state: %#v", diags)
			}
			if state.ResultJSON.ValueString() != tt.want {
				t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), tt.want)
			}
		})
	}

	wantPaths := []string{
		"/management/organizations/DEFAULT/environments/DEFAULT/members/permissions",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/members/permissions",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/applications/app-1/members/permissions",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/protected-resources/resource-1/members/permissions",
	}
	if len(paths) != len(wantPaths) {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
	for i := range wantPaths {
		if paths[i] != wantPaths[i] {
			t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
		}
	}
}

func TestPermissionsMetadataReadValidatesRequestShapeBeforeHTTP(t *testing.T) {
	dataSource := &PermissionsMetadataDataSource{}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []PermissionsMetadataModel{
		{Kind: types.StringValue("unknown")},
		{Kind: types.StringValue("application_member_permissions"), ApplicationID: types.StringValue("app-1")},
		{Kind: types.StringValue("application_member_permissions"), DomainID: types.StringValue("domain-1")},
		{Kind: types.StringValue("domain_member_permissions")},
		{Kind: types.StringValue("protected_resource_member_permissions"), DomainID: types.StringValue("domain-1")},
	}

	for _, model := range cases {
		config := permissionsMetadataConfig(schemaResp.Schema, model)
		readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
		if !readResp.Diagnostics.HasError() {
			t.Fatalf("expected diagnostics for model %#v", model)
		}
	}
}

func TestPermissionsMetadataReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/members/permissions", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "permissions failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &PermissionsMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := permissionsMetadataConfig(schemaResp.Schema, PermissionsMetadataModel{
		Kind: types.StringValue("environment_member_permissions"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func permissionsMetadataConfig(schema datasourceschema.Schema, model PermissionsMetadataModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"kind":                  tftypes.String,
				"domain_id":             tftypes.String,
				"application_id":        tftypes.String,
				"protected_resource_id": tftypes.String,
				"result_json":           tftypes.String,
			}},
			map[string]tftypes.Value{
				"kind":                  tftypes.NewValue(tftypes.String, model.Kind.ValueString()),
				"domain_id":             permissionsMetadataStringConfigValue(model.DomainID),
				"application_id":        permissionsMetadataStringConfigValue(model.ApplicationID),
				"protected_resource_id": permissionsMetadataStringConfigValue(model.ProtectedResourceID),
				"result_json":           tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}

func permissionsMetadataStringConfigValue(value types.String) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value.ValueString())
}
