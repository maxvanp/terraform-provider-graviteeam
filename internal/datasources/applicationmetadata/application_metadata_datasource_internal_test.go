package applicationmetadata

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestApplicationMetadataNewMetadataAndConfigure(t *testing.T) {
	t.Parallel()

	dataSource, ok := NewApplicationMetadataDataSource().(*ApplicationMetadataDataSource)
	if !ok {
		t.Fatalf("data source type = %T, want *ApplicationMetadataDataSource", NewApplicationMetadataDataSource())
	}

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &metadataResp)
	if got, want := metadataResp.TypeName, "graviteeam_application_metadata"; got != want {
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

func TestApplicationMetadataConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&ApplicationMetadataDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestApplicationMetadataConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	dataSource := &ApplicationMetadataDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestApplicationMetadataReadResolvesSupportedKinds(t *testing.T) {
	var queries []map[string]string
	var paths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/analytics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		queries = append(queries, firstQueryValues(r.URL.Query()))
		_, _ = w.Write([]byte(`{"count":2}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/resources", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		queries = append(queries, firstQueryValues(r.URL.Query()))
		_, _ = w.Write([]byte(`{"resources":[{"id":"resource-1"}]}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/resources/resource%2Fid%20with%20spaces/policies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.EscapedPath())
		queries = append(queries, firstQueryValues(r.URL.Query()))
		_, _ = w.Write([]byte(`["policy-1"]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &ApplicationMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []struct {
		name  string
		model ApplicationMetadataModel
		want  string
	}{
		{
			name: "analytics",
			model: ApplicationMetadataModel{
				DomainID:      types.StringValue("domain-123"),
				ApplicationID: types.StringValue("app-123"),
				Kind:          types.StringValue("analytics"),
				Type:          types.StringValue("DATE_HISTO"),
				Field:         types.StringValue("status"),
				From:          types.Int64Value(1000),
				To:            types.Int64Value(2000),
				Interval:      types.Int64Value(100),
				Size:          types.Int64Value(20),
			},
			want: "{\n  \"count\": 2\n}",
		},
		{
			name: "resources",
			model: ApplicationMetadataModel{
				DomainID:      types.StringValue("domain-123"),
				ApplicationID: types.StringValue("app-123"),
				Kind:          types.StringValue("resources"),
				Size:          types.Int64Value(25),
			},
			want: "{\n  \"resources\": [\n    {\n      \"id\": \"resource-1\"\n    }\n  ]\n}",
		},
		{
			name: "resource policies",
			model: ApplicationMetadataModel{
				DomainID:      types.StringValue("domain-123"),
				ApplicationID: types.StringValue("app-123"),
				ResourceID:    types.StringValue("resource/id with spaces"),
				Kind:          types.StringValue("resource_policies"),
			},
			want: "[\n  \"policy-1\"\n]",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			config := applicationMetadataConfig(schemaResp.Schema, tt.model)
			readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

			dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
			}
			var state ApplicationMetadataModel
			if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("get state: %#v", diags)
			}
			if state.ResultJSON.ValueString() != tt.want {
				t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), tt.want)
			}
		})
	}

	wantPaths := []string{
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/analytics",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/resources",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/resources/resource%2Fid%20with%20spaces/policies",
	}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
	wantQueries := []map[string]string{
		{
			"field":    "status",
			"from":     "1000",
			"interval": "100",
			"size":     "20",
			"to":       "2000",
			"type":     "DATE_HISTO",
		},
		{
			"page": "0",
			"size": "25",
		},
		{},
	}
	if !reflect.DeepEqual(queries, wantQueries) {
		t.Fatalf("queries = %#v, want %#v", queries, wantQueries)
	}
}

func TestApplicationMetadataReadValidatesRequestShapeBeforeHTTP(t *testing.T) {
	dataSource := &ApplicationMetadataDataSource{}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []ApplicationMetadataModel{
		{
			DomainID:      types.StringValue("domain-123"),
			ApplicationID: types.StringValue("app-123"),
			Kind:          types.StringValue("unknown"),
		},
		{
			DomainID:      types.StringValue("domain-123"),
			ApplicationID: types.StringValue("app-123"),
			Kind:          types.StringValue("resource_policies"),
		},
	}

	for _, model := range cases {
		config := applicationMetadataConfig(schemaResp.Schema, model)
		readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
		if !readResp.Diagnostics.HasError() {
			t.Fatalf("expected diagnostics for model %#v", model)
		}
	}
}

func TestApplicationMetadataReadReportsInvalidConfig(t *testing.T) {
	t.Parallel()

	dataSource := &ApplicationMetadataDataSource{
		client: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id":      tftypes.Number,
				"application_id": tftypes.String,
				"resource_id":    tftypes.String,
				"kind":           tftypes.String,
				"type":           tftypes.String,
				"field":          tftypes.String,
				"from":           tftypes.Number,
				"to":             tftypes.Number,
				"interval":       tftypes.Number,
				"size":           tftypes.Number,
				"result_json":    tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id":      tftypes.NewValue(tftypes.Number, 123),
				"application_id": tftypes.NewValue(tftypes.String, "app-123"),
				"resource_id":    tftypes.NewValue(tftypes.String, nil),
				"kind":           tftypes.NewValue(tftypes.String, "analytics"),
				"type":           tftypes.NewValue(tftypes.String, nil),
				"field":          tftypes.NewValue(tftypes.String, nil),
				"from":           tftypes.NewValue(tftypes.Number, nil),
				"to":             tftypes.NewValue(tftypes.Number, nil),
				"interval":       tftypes.NewValue(tftypes.Number, nil),
				"size":           tftypes.NewValue(tftypes.Number, nil),
				"result_json":    tftypes.NewValue(tftypes.String, nil),
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

func TestApplicationMetadataReadTreatsEmptyAnalyticsErrorsAsEmptyResult(t *testing.T) {
	tests := map[string]struct {
		status int
		body   string
	}{
		"server error":   {status: http.StatusInternalServerError, body: "no analytics yet"},
		"malformed json": {status: http.StatusBadRequest, body: "Malformed json request"},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/analytics", func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, tt.body, tt.status)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			dataSource := &ApplicationMetadataDataSource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp datasource.SchemaResponse
			dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
			config := applicationMetadataConfig(schemaResp.Schema, ApplicationMetadataModel{
				DomainID:      types.StringValue("domain-123"),
				ApplicationID: types.StringValue("app-123"),
				Kind:          types.StringValue("analytics"),
				From:          types.Int64Value(1000),
				To:            types.Int64Value(2000),
			})

			readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
			}
			var state ApplicationMetadataModel
			if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("get state: %#v", diags)
			}
			if state.ResultJSON.ValueString() != "{}" {
				t.Fatalf("result_json = %q, want {}", state.ResultJSON.ValueString())
			}
		})
	}
}

func TestApplicationMetadataReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/resources", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "resources failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &ApplicationMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := applicationMetadataConfig(schemaResp.Schema, ApplicationMetadataModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Kind:          types.StringValue("resources"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
}

func firstQueryValues(values url.Values) map[string]string {
	result := map[string]string{}
	for key, value := range values {
		if len(value) > 0 {
			result[key] = value[0]
		}
	}
	return result
}

func applicationMetadataConfig(schema datasourceschema.Schema, model ApplicationMetadataModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id":      tftypes.String,
				"application_id": tftypes.String,
				"resource_id":    tftypes.String,
				"kind":           tftypes.String,
				"type":           tftypes.String,
				"field":          tftypes.String,
				"from":           tftypes.Number,
				"to":             tftypes.Number,
				"interval":       tftypes.Number,
				"size":           tftypes.Number,
				"result_json":    tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id":      tftypes.NewValue(tftypes.String, model.DomainID.ValueString()),
				"application_id": tftypes.NewValue(tftypes.String, model.ApplicationID.ValueString()),
				"resource_id":    applicationMetadataStringConfigValue(model.ResourceID),
				"kind":           tftypes.NewValue(tftypes.String, model.Kind.ValueString()),
				"type":           applicationMetadataStringConfigValue(model.Type),
				"field":          applicationMetadataStringConfigValue(model.Field),
				"from":           applicationMetadataInt64ConfigValue(model.From),
				"to":             applicationMetadataInt64ConfigValue(model.To),
				"interval":       applicationMetadataInt64ConfigValue(model.Interval),
				"size":           applicationMetadataInt64ConfigValue(model.Size),
				"result_json":    tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}

func applicationMetadataStringConfigValue(value types.String) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value.ValueString())
}

func applicationMetadataInt64ConfigValue(value types.Int64) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.Number, nil)
	}
	return tftypes.NewValue(tftypes.Number, big.NewFloat(float64(value.ValueInt64())))
}
