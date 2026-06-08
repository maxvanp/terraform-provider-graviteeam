package flows

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

func TestFlowsMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewFlowsDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_flows" {
		t.Fatalf("type name = %q, want graviteeam_flows", resp.TypeName)
	}
}

func TestFlowsSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewFlowsDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["domain_id"]; !attr.IsRequired() {
		t.Fatal("domain_id should be required")
	}
	if attr := resp.Schema.Attributes["flows"]; !attr.IsComputed() {
		t.Fatal("flows should be computed")
	}
}

func TestFlowsConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	ds := &FlowsDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestFlowsConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	ds := &FlowsDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if ds.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestFlowsConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &FlowsDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if ds.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestFormatFlowsProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got := formatFlows([]interface{}{
		map[string]interface{}{"id": "login", "enabled": true},
	})
	want := "[\n  {\n    \"enabled\": true,\n    \"id\": \"login\"\n  }\n]"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFlowsReadFormatsRemoteFlows(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/flows", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`[{"id":"login","enabled":true}]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &FlowsDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := flowsConfig(schemaResp.Schema, FlowsModel{
		DomainID: types.StringValue("domain-123"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var state FlowsModel
	if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	want := "[\n  {\n    \"enabled\": true,\n    \"id\": \"login\"\n  }\n]"
	if state.Flows.ValueString() != want {
		t.Fatalf("flows = %q, want %q", state.Flows.ValueString(), want)
	}
}

func TestFlowsReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/flows", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "flows failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &FlowsDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := flowsConfig(schemaResp.Schema, FlowsModel{
		DomainID: types.StringValue("domain-123"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func TestFlowsReadReportsInvalidConfig(t *testing.T) {
	dataSource := &FlowsDataSource{}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.Number,
				"flows":     tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.Number, 123),
				"flows":     tftypes.NewValue(tftypes.String, nil),
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

func flowsConfig(schema datasourceschema.Schema, model FlowsModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.String,
				"flows":     tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.String, model.DomainID.ValueString()),
				"flows":     tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}
