package adminmetadata

import (
	"context"
	"math/big"
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

func TestAdminMetadataReadResolvesEnvironmentAndOrganizationScopes(t *testing.T) {
	var paths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/_hrid/domain%2Fhr%20id", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.EscapedPath())
		_, _ = w.Write([]byte(`{"id":"domain-123"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/audits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.EscapedPath()+"?"+r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"data":[{"id":"audit-1"}]}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain%2F1/users/user@example.com/audits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.EscapedPath()+"?"+r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"data":[{"id":"user-audit-1"}]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &AdminMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []struct {
		name     string
		model    AdminMetadataModel
		wantJSON string
		wantSize int64
	}{
		{
			name: "domain by hrid",
			model: AdminMetadataModel{
				Kind: types.StringValue("domain_by_hrid"),
				HRID: types.StringValue("domain/hr id"),
			},
			wantJSON: "{\n  \"id\": \"domain-123\"\n}",
			wantSize: 10,
		},
		{
			name: "organization audits default size",
			model: AdminMetadataModel{
				Kind: types.StringValue("organization_audits"),
			},
			wantJSON: "{\n  \"data\": [\n    {\n      \"id\": \"audit-1\"\n    }\n  ]\n}",
			wantSize: 10,
		},
		{
			name: "user audits explicit size",
			model: AdminMetadataModel{
				Kind:     types.StringValue("user_audits"),
				DomainID: types.StringValue("domain/1"),
				UserID:   types.StringValue("user@example.com"),
				Size:     types.Int64Value(25),
			},
			wantJSON: "{\n  \"data\": [\n    {\n      \"id\": \"user-audit-1\"\n    }\n  ]\n}",
			wantSize: 25,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			config := adminMetadataConfig(schemaResp.Schema, tt.model)
			readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

			dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
			}
			var state AdminMetadataModel
			if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("get state: %#v", diags)
			}
			if state.ResultJSON.ValueString() != tt.wantJSON {
				t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), tt.wantJSON)
			}
			if state.Size.ValueInt64() != tt.wantSize {
				t.Fatalf("size = %d, want %d", state.Size.ValueInt64(), tt.wantSize)
			}
		})
	}

	wantPaths := []string{
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/_hrid/domain%2Fhr%20id",
		"/management/organizations/DEFAULT/audits?page=0&size=10",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain%2F1/users/user@example.com/audits?page=0&size=25",
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

func TestAdminMetadataReadValidatesRequestShapeBeforeHTTP(t *testing.T) {
	dataSource := &AdminMetadataDataSource{}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []AdminMetadataModel{
		{Kind: types.StringValue("unknown")},
		{Kind: types.StringValue("domain_by_hrid")},
		{Kind: types.StringValue("user_audits"), UserID: types.StringValue("user-1")},
		{Kind: types.StringValue("user_audits"), DomainID: types.StringValue("domain-1")},
	}

	for _, model := range cases {
		config := adminMetadataConfig(schemaResp.Schema, model)
		readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
		if !readResp.Diagnostics.HasError() {
			t.Fatalf("expected diagnostics for model %#v", model)
		}
	}
}

func TestAdminMetadataReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/_hrid/domain-1", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "metadata failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &AdminMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := adminMetadataConfig(schemaResp.Schema, AdminMetadataModel{
		Kind: types.StringValue("domain_by_hrid"),
		HRID: types.StringValue("domain-1"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func adminMetadataConfig(schema datasourceschema.Schema, model AdminMetadataModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"kind":        tftypes.String,
				"domain_id":   tftypes.String,
				"user_id":     tftypes.String,
				"hrid":        tftypes.String,
				"size":        tftypes.Number,
				"result_json": tftypes.String,
			}},
			map[string]tftypes.Value{
				"kind":        tftypes.NewValue(tftypes.String, model.Kind.ValueString()),
				"domain_id":   adminMetadataStringConfigValue(model.DomainID),
				"user_id":     adminMetadataStringConfigValue(model.UserID),
				"hrid":        adminMetadataStringConfigValue(model.HRID),
				"size":        adminMetadataInt64ConfigValue(model.Size),
				"result_json": tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}

func adminMetadataStringConfigValue(value types.String) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value.ValueString())
}

func adminMetadataInt64ConfigValue(value types.Int64) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.Number, nil)
	}
	return tftypes.NewValue(tftypes.Number, big.NewFloat(float64(value.ValueInt64())))
}
