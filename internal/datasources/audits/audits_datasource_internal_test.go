package audits

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestAuditsNewMetadataAndConfigure(t *testing.T) {
	t.Parallel()

	dataSource, ok := NewAuditsDataSource().(*AuditsDataSource)
	if !ok {
		t.Fatalf("data source type = %T, want *AuditsDataSource", NewAuditsDataSource())
	}

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &metadataResp)
	if got, want := metadataResp.TypeName, "graviteeam_audits"; got != want {
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

func TestAuditsConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&AuditsDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestAuditsConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	dataSource := &AuditsDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestAuditSizeUsesDefaultAndExplicitValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		model AuditsModel
		want  int
	}{
		"default null":    {AuditsModel{Size: types.Int64Null()}, 10},
		"default unknown": {AuditsModel{Size: types.Int64Unknown()}, 10},
		"explicit":        {AuditsModel{Size: types.Int64Value(5)}, 5},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := auditSize(test.model); got != test.want {
				t.Fatalf("size = %d, want %d", got, test.want)
			}
		})
	}
}

func TestExtractAuditEntriesReturnsDataSliceOnly(t *testing.T) {
	t.Parallel()

	entries := []interface{}{
		map[string]interface{}{"id": "audit-1"},
	}

	if got := extractAuditEntries(map[string]interface{}{"data": entries}); !reflect.DeepEqual(got, entries) {
		t.Fatalf("entries = %#v, want %#v", got, entries)
	}
	if got := extractAuditEntries(map[string]interface{}{"data": "not-a-list"}); got != nil {
		t.Fatalf("entries = %#v, want nil", got)
	}
	if got := extractAuditEntries(map[string]interface{}{}); got != nil {
		t.Fatalf("entries = %#v, want nil", got)
	}
}

func TestFormatAuditEntriesProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatAuditEntries(map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"id": "audit-1", "type": "USER_CREATED"},
		},
	})
	if err != nil {
		t.Fatalf("format audits: %v", err)
	}
	want := "[\n  {\n    \"id\": \"audit-1\",\n    \"type\": \"USER_CREATED\"\n  }\n]"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFormatAuditEntriesUsesNullWhenDataIsMissing(t *testing.T) {
	t.Parallel()

	got, err := formatAuditEntries(map[string]interface{}{})
	if err != nil {
		t.Fatalf("format audits: %v", err)
	}
	if got != "null" {
		t.Fatalf("json = %q, want null", got)
	}
}

func TestFormatAuditEntriesReportsMarshalError(t *testing.T) {
	t.Parallel()

	_, err := formatAuditEntries(map[string]interface{}{
		"data": []interface{}{func() {}},
	})
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestAuditsReadReportsInvalidConfig(t *testing.T) {
	t.Parallel()

	dataSource := &AuditsDataSource{
		client: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.Number,
				"size":      tftypes.Number,
				"audits":    tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.Number, big.NewFloat(123)),
				"size":      tftypes.NewValue(tftypes.Number, big.NewFloat(5)),
				"audits":    tftypes.NewValue(tftypes.String, nil),
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

func TestAuditsReadUsesDefaultSizeAndFormatsEntries(t *testing.T) {
	var rawQuery string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/audits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		rawQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "audit-1", "type": "USER_CREATED"},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &AuditsDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := auditsConfig(t, schemaResp.Schema, AuditsModel{
		DomainID: types.StringValue("domain-123"),
		Size:     types.Int64Null(),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var state AuditsModel
	if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	if rawQuery != "page=0&size=10" {
		t.Fatalf("query = %q, want page=0&size=10", rawQuery)
	}
	if state.Size.ValueInt64() != 10 {
		t.Fatalf("size = %d, want 10", state.Size.ValueInt64())
	}
	want := "[\n  {\n    \"id\": \"audit-1\",\n    \"type\": \"USER_CREATED\"\n  }\n]"
	if state.Audits.ValueString() != want {
		t.Fatalf("audits = %q, want %q", state.Audits.ValueString(), want)
	}
}

func TestAuditsReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/audits", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "audits failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &AuditsDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := auditsConfig(t, schemaResp.Schema, AuditsModel{
		DomainID: types.StringValue("domain-123"),
		Size:     types.Int64Value(5),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func auditsConfig(t *testing.T, schema datasourceschema.Schema, model AuditsModel) tfsdk.Config {
	t.Helper()

	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"domain_id": tftypes.String,
			"size":      tftypes.Number,
			"audits":    tftypes.String,
		}},
		map[string]tftypes.Value{
			"domain_id": tftypes.NewValue(tftypes.String, model.DomainID.ValueString()),
			"size":      int64ConfigValue(model.Size),
			"audits":    tftypes.NewValue(tftypes.String, nil),
		},
	)

	return tfsdk.Config{
		Raw:    raw,
		Schema: schema,
	}
}

func int64ConfigValue(value types.Int64) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.Number, nil)
	}
	return tftypes.NewValue(tftypes.Number, big.NewFloat(float64(value.ValueInt64())))
}
