package selfmetadata

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

func TestSelfMetadataConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	dataSource := &SelfMetadataDataSource{}
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

func TestSelfMetadataConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	dataSource := &SelfMetadataDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestSelfMetadataReadResolvesSupportedKinds(t *testing.T) {
	var paths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`{"id":"admin"}`))
	})
	mux.HandleFunc("/management/user/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`{"notifications":[{"id":"n1"}]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &SelfMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []struct {
		name  string
		model SelfMetadataModel
		want  string
	}{
		{
			name: "current user",
			model: SelfMetadataModel{
				Kind: types.StringValue("current_user"),
			},
			want: "{\n  \"id\": \"admin\"\n}",
		},
		{
			name: "notifications",
			model: SelfMetadataModel{
				Kind: types.StringValue("notifications"),
			},
			want: "{\n  \"notifications\": [\n    {\n      \"id\": \"n1\"\n    }\n  ]\n}",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			config := selfMetadataConfig(schemaResp.Schema, tt.model)
			readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

			dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
			}
			var state SelfMetadataModel
			if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("get state: %#v", diags)
			}
			if state.ResultJSON.ValueString() != tt.want {
				t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), tt.want)
			}
		})
	}

	wantPaths := []string{
		"/management/user",
		"/management/user/notifications",
	}
	if strings.Join(paths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
}

func TestSelfMetadataReadValidatesKindAndReportsRemoteError(t *testing.T) {
	dataSource := &SelfMetadataDataSource{}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := selfMetadataConfig(schemaResp.Schema, SelfMetadataModel{
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
	mux.HandleFunc("/management/user/notifications", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "self metadata failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	remoteDataSource := &SelfMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	config = selfMetadataConfig(schemaResp.Schema, SelfMetadataModel{
		Kind: types.StringValue("notifications"),
	})
	readResp = &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	remoteDataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func TestSelfMetadataReadReportsInvalidConfig(t *testing.T) {
	t.Parallel()

	dataSource := &SelfMetadataDataSource{
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

func selfMetadataConfig(schema datasourceschema.Schema, model SelfMetadataModel) tfsdk.Config {
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
