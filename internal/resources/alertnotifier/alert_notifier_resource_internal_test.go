package alertnotifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestAlertNotifierMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewAlertNotifierResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_alert_notifier" {
		t.Fatalf("type name = %q, want graviteeam_alert_notifier", resp.TypeName)
	}
}

func TestAlertNotifierSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewAlertNotifierResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "type", "configuration"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["enabled"]; !attr.IsOptional() || !attr.IsComputed() {
		t.Fatalf("enabled should be optional+computed, got optional=%t computed=%t", attr.IsOptional(), attr.IsComputed())
	}
	if attr := resp.Schema.Attributes["id"]; !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
}

func TestAlertNotifierConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &AlertNotifierResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildCreateBodyIncludesImmutableAndMutableFields(t *testing.T) {
	t.Parallel()

	model := AlertNotifierModel{
		Name:          types.StringValue("webhook"),
		Type:          types.StringValue("webhook-notifier"),
		Configuration: types.StringValue(`{"url":"https://example.test","method":"POST"}`),
		Enabled:       types.BoolValue(true),
	}

	body := buildCreateBody(model)

	if body["name"] != "webhook" {
		t.Fatalf("name = %#v, want webhook", body["name"])
	}
	if body["type"] != "webhook-notifier" {
		t.Fatalf("type = %#v, want webhook-notifier", body["type"])
	}
	if body["configuration"] != `{"url":"https://example.test","method":"POST"}` {
		t.Fatalf("configuration = %#v, want original JSON", body["configuration"])
	}
	if body["enabled"] != true {
		t.Fatalf("enabled = %#v, want true", body["enabled"])
	}
}

func TestBuildPatchBodyOmitsImmutableType(t *testing.T) {
	t.Parallel()

	model := AlertNotifierModel{
		Name:          types.StringValue("webhook"),
		Type:          types.StringValue("webhook-notifier"),
		Configuration: types.StringValue(`{"url":"https://example.test"}`),
		Enabled:       types.BoolValue(false),
	}

	body := buildPatchBody(model)

	if _, ok := body["type"]; ok {
		t.Fatalf("type = %#v, want omitted for PATCH", body["type"])
	}
	if body["name"] != "webhook" {
		t.Fatalf("name = %#v, want webhook", body["name"])
	}
	if body["configuration"] != `{"url":"https://example.test"}` {
		t.Fatalf("configuration = %#v, want original JSON", body["configuration"])
	}
	if body["enabled"] != false {
		t.Fatalf("enabled = %#v, want false", body["enabled"])
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := AlertNotifierModel{
		Name:          types.StringValue("old"),
		Type:          types.StringValue("webhook-notifier"),
		Configuration: types.StringValue(`{"secret":"plain"}`),
		Enabled:       types.BoolValue(false),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new",
		"type":          "webhook-notifier",
		"configuration": `{"secret":"***"}`,
		"enabled":       true,
	})

	if model.Name.ValueString() != "new" {
		t.Fatalf("name = %q, want new", model.Name.ValueString())
	}
	if model.Type.ValueString() != "webhook-notifier" {
		t.Fatalf("type = %q, want webhook-notifier", model.Type.ValueString())
	}
	if model.Configuration.ValueString() != `{"secret":"plain"}` {
		t.Fatalf("configuration = %q, want preserved unmasked state", model.Configuration.ValueString())
	}
	if !model.Enabled.ValueBool() {
		t.Fatal("enabled = false, want true")
	}
}

func TestAlertNotifierCRUDPreservesPlannedConfiguration(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	var bodies []map[string]interface{}
	var methods []string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/notifiers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected alert notifier collection method %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		methods = append(methods, "create")
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "notifier-123"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/notifiers/notifier-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "notifier-123",
				"name":          "webhook",
				"type":          "webhook-notifier",
				"configuration": `{"secret":"***"}`,
				"enabled":       true,
			})
		case http.MethodPatch:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode patch body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "notifier-123"})
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected alert notifier item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &AlertNotifierResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := alertNotifierPlan(t, schemaResp.Schema, AlertNotifierModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("webhook"),
		Type:          types.StringValue("webhook-notifier"),
		Configuration: types.StringValue(`{"secret":"plain"}`),
		Enabled:       types.BoolValue(true),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState AlertNotifierModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := readState.Configuration.ValueString(), `{"secret":"plain"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}

	updatePlan := alertNotifierPlan(t, schemaResp.Schema, AlertNotifierModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("webhook updated"),
		Type:          types.StringValue("webhook-notifier"),
		Configuration: types.StringValue(`{"secret":"updated"}`),
		Enabled:       types.BoolValue(false),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: readResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if _, ok := bodies[1]["type"]; ok {
		t.Fatalf("patch body should omit immutable type: %#v", bodies[1])
	}
	if got, want := bodies[1]["configuration"], `{"secret":"updated"}`; got != want {
		t.Fatalf("patch configuration = %#v, want %#v", got, want)
	}
	if want := []string{"create", "read", "update", "delete"}; !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
}

func TestAlertNotifierReadRemovesMissingNotifierAndDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/notifiers/notifier-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &AlertNotifierResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &AlertNotifierModel{
		ID:            types.StringValue("notifier-123"),
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("webhook"),
		Type:          types.StringValue("webhook-notifier"),
		Configuration: types.StringValue(`{"secret":"plain"}`),
		Enabled:       types.BoolValue(true),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing alert notifier to remove state, got %#v", readResp.State.Raw)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func alertNotifierPlan(t *testing.T, schema resourceschema.Schema, model AlertNotifierModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
