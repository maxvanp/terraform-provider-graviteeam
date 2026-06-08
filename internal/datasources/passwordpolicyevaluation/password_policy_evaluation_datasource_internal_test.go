package passwordpolicyevaluation

import (
	"context"
	"encoding/json"
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

func TestPasswordPolicyEvaluationNewMetadataAndConfigure(t *testing.T) {
	t.Parallel()

	dataSource, ok := NewPasswordPolicyEvaluationDataSource().(*PasswordPolicyEvaluationDataSource)
	if !ok {
		t.Fatalf("data source type = %T, want *PasswordPolicyEvaluationDataSource", NewPasswordPolicyEvaluationDataSource())
	}

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &metadataResp)
	if got, want := metadataResp.TypeName, "graviteeam_password_policy_evaluation"; got != want {
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

func TestFormatJSONHandlesEmptyJSONAndText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty": {nil, "null"},
		"text":  {[]byte("not json"), `"not json"`},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := formatJSON(test.raw)
			if err != nil {
				t.Fatalf("format json: %v", err)
			}
			if got != test.want {
				t.Fatalf("json = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatJSONProducesStableIndentedEvaluationJSON(t *testing.T) {
	t.Parallel()

	got, err := formatJSON([]byte(`{"valid":false,"errors":["too_short"]}`))
	if err != nil {
		t.Fatalf("format json: %v", err)
	}
	want := "{\n  \"errors\": [\n    \"too_short\"\n  ],\n  \"valid\": false\n}"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestPasswordPolicyEvaluationReadPostsPasswordAndUserContext(t *testing.T) {
	var body map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123/evaluate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"valid":false,"errors":["too_short"]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &PasswordPolicyEvaluationDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := passwordPolicyEvaluationConfig(t, schemaResp.Schema, PasswordPolicyEvaluationModel{
		DomainID: types.StringValue("domain-123"),
		PolicyID: types.StringValue("policy-123"),
		Password: types.StringValue("short"),
		UserID:   types.StringValue("user-123"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var state PasswordPolicyEvaluationModel
	if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	if got := body["password"]; got != "short" {
		t.Fatalf("password body = %#v", body)
	}
	if got := body["userId"]; got != "user-123" {
		t.Fatalf("userId body = %#v", body)
	}
	want := "{\n  \"errors\": [\n    \"too_short\"\n  ],\n  \"valid\": false\n}"
	if state.ResultJSON.ValueString() != want {
		t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), want)
	}
}

func TestPasswordPolicyEvaluationReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123/evaluate", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "evaluation failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &PasswordPolicyEvaluationDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := passwordPolicyEvaluationConfig(t, schemaResp.Schema, PasswordPolicyEvaluationModel{
		DomainID: types.StringValue("domain-123"),
		PolicyID: types.StringValue("policy-123"),
		Password: types.StringValue("short"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func passwordPolicyEvaluationConfig(t *testing.T, schema datasourceschema.Schema, model PasswordPolicyEvaluationModel) tfsdk.Config {
	t.Helper()

	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"domain_id":   tftypes.String,
			"policy_id":   tftypes.String,
			"password":    tftypes.String,
			"user_id":     tftypes.String,
			"result_json": tftypes.String,
		}},
		map[string]tftypes.Value{
			"domain_id":   tftypes.NewValue(tftypes.String, model.DomainID.ValueString()),
			"policy_id":   tftypes.NewValue(tftypes.String, model.PolicyID.ValueString()),
			"password":    tftypes.NewValue(tftypes.String, model.Password.ValueString()),
			"user_id":     stringConfigValue(model.UserID),
			"result_json": tftypes.NewValue(tftypes.String, nil),
		},
	)

	return tfsdk.Config{
		Raw:    raw,
		Schema: schema,
	}
}

func stringConfigValue(value types.String) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value.ValueString())
}
