package analytics

import (
	"context"
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

func TestAnalyticsNewMetadataAndConfigure(t *testing.T) {
	t.Parallel()

	dataSource, ok := NewAnalyticsDataSource().(*AnalyticsDataSource)
	if !ok {
		t.Fatalf("data source type = %T, want *AnalyticsDataSource", NewAnalyticsDataSource())
	}

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &metadataResp)
	if got, want := metadataResp.TypeName, "graviteeam_analytics"; got != want {
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

func TestAnalyticsConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&AnalyticsDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestAnalyticsConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	dataSource := &AnalyticsDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestBuildAnalyticsParamsUsesExplicitValues(t *testing.T) {
	t.Parallel()

	model := AnalyticsModel{
		Type:     types.StringValue("GROUP_BY"),
		Field:    types.StringValue("type"),
		From:     types.Int64Value(1000),
		To:       types.Int64Value(2000),
		Interval: types.Int64Value(300),
		Size:     types.Int64Value(5),
	}

	got := buildAnalyticsParams(model, 9999)
	want := map[string]string{
		"type":     "GROUP_BY",
		"field":    "type",
		"from":     "1000",
		"to":       "2000",
		"interval": "300",
		"size":     "5",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("params = %#v, want %#v", got, want)
	}
}

func TestBuildAnalyticsParamsUsesDefaultTimeWindowAndOmitsUnknownOptionals(t *testing.T) {
	t.Parallel()

	model := AnalyticsModel{
		Type:     types.StringValue("COUNT"),
		Field:    types.StringUnknown(),
		From:     types.Int64Null(),
		To:       types.Int64Unknown(),
		Interval: types.Int64Null(),
		Size:     types.Int64Unknown(),
	}

	got := buildAnalyticsParams(model, 172800000)
	want := map[string]string{
		"type": "COUNT",
		"from": "86400000",
		"to":   "172800000",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("params = %#v, want %#v", got, want)
	}
}

func TestFormatAnalyticsResultProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got := formatAnalyticsResult(map[string]interface{}{
		"count": float64(2),
		"type":  "COUNT",
	})
	want := "{\n  \"count\": 2,\n  \"type\": \"COUNT\"\n}"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestAnalyticsReadFormatsRemoteResult(t *testing.T) {
	var queryValues map[string]string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/analytics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		queryValues = map[string]string{}
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				queryValues[key] = values[0]
			}
		}
		_, _ = w.Write([]byte(`{"count":2,"type":"COUNT"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &AnalyticsDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := analyticsConfig(schemaResp.Schema, AnalyticsModel{
		DomainID: types.StringValue("domain-123"),
		Type:     types.StringValue("COUNT"),
		Field:    types.StringValue("type"),
		From:     types.Int64Value(1000),
		To:       types.Int64Value(2000),
		Interval: types.Int64Value(300),
		Size:     types.Int64Value(5),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var state AnalyticsModel
	if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	wantQuery := map[string]string{
		"type":     "COUNT",
		"field":    "type",
		"from":     "1000",
		"to":       "2000",
		"interval": "300",
		"size":     "5",
	}
	if !reflect.DeepEqual(queryValues, wantQuery) {
		t.Fatalf("query = %#v, want %#v", queryValues, wantQuery)
	}
	want := "{\n  \"count\": 2,\n  \"type\": \"COUNT\"\n}"
	if state.Result.ValueString() != want {
		t.Fatalf("result = %q, want %q", state.Result.ValueString(), want)
	}
}

func TestAnalyticsReadTreatsRemote500AsEmptyResult(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/analytics", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no analytics yet", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &AnalyticsDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := analyticsConfig(schemaResp.Schema, AnalyticsModel{
		DomainID: types.StringValue("domain-123"),
		Type:     types.StringValue("COUNT"),
		From:     types.Int64Value(1000),
		To:       types.Int64Value(2000),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var state AnalyticsModel
	if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	if state.Result.ValueString() != "{}" {
		t.Fatalf("result = %q, want {}", state.Result.ValueString())
	}
}

func TestAnalyticsReadReportsNon500RemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/analytics", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "analytics failed", http.StatusBadRequest)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &AnalyticsDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := analyticsConfig(schemaResp.Schema, AnalyticsModel{
		DomainID: types.StringValue("domain-123"),
		Type:     types.StringValue("COUNT"),
		From:     types.Int64Value(1000),
		To:       types.Int64Value(2000),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func TestAnalyticsReadReportsInvalidConfig(t *testing.T) {
	t.Parallel()

	dataSource := &AnalyticsDataSource{
		client: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.Number,
				"type":      tftypes.String,
				"field":     tftypes.String,
				"from":      tftypes.Number,
				"to":        tftypes.Number,
				"interval":  tftypes.Number,
				"size":      tftypes.Number,
				"result":    tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.Number, 123),
				"type":      tftypes.NewValue(tftypes.String, "COUNT"),
				"field":     tftypes.NewValue(tftypes.String, nil),
				"from":      tftypes.NewValue(tftypes.Number, big.NewFloat(1000)),
				"to":        tftypes.NewValue(tftypes.Number, big.NewFloat(2000)),
				"interval":  tftypes.NewValue(tftypes.Number, nil),
				"size":      tftypes.NewValue(tftypes.Number, nil),
				"result":    tftypes.NewValue(tftypes.String, nil),
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

func analyticsConfig(schema datasourceschema.Schema, model AnalyticsModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.String,
				"type":      tftypes.String,
				"field":     tftypes.String,
				"from":      tftypes.Number,
				"to":        tftypes.Number,
				"interval":  tftypes.Number,
				"size":      tftypes.Number,
				"result":    tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.String, model.DomainID.ValueString()),
				"type":      tftypes.NewValue(tftypes.String, model.Type.ValueString()),
				"field":     analyticsStringConfigValue(model.Field),
				"from":      analyticsInt64ConfigValue(model.From),
				"to":        analyticsInt64ConfigValue(model.To),
				"interval":  analyticsInt64ConfigValue(model.Interval),
				"size":      analyticsInt64ConfigValue(model.Size),
				"result":    tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}

func analyticsStringConfigValue(value types.String) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value.ValueString())
}

func analyticsInt64ConfigValue(value types.Int64) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.Number, nil)
	}
	return tftypes.NewValue(tftypes.Number, big.NewFloat(float64(value.ValueInt64())))
}
