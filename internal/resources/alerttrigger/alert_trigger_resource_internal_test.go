package alerttrigger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewAlertTriggerResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_alert_trigger"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewAlertTriggerResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "type"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"enabled", "alert_notifier_ids"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Fatalf("attribute %q should be optional+computed", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; attr == nil || !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&AlertTriggerResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&AlertTriggerResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestValidateTriggerTypeNormalizesAcceptedValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"too_many_login_failures", "TOO_MANY_LOGIN_FAILURES", "risk_assessment", "RISK_ASSESSMENT"} {
		if err := validateTriggerType(value); err != nil {
			t.Fatalf("validateTriggerType(%q) returned error: %v", value, err)
		}
	}
}

func TestValidateTriggerTypeRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	if err := validateTriggerType("unknown"); err == nil {
		t.Fatal("validateTriggerType(unknown) returned nil, want error")
	}
}

func TestFindTriggerMatchesNormalizedType(t *testing.T) {
	triggers := []map[string]interface{}{
		{"type": "RISK_ASSESSMENT", "enabled": false},
		{"type": "too_many_login_failures", "enabled": true},
	}

	trigger, ok := findTrigger(triggers, "TOO_MANY_LOGIN_FAILURES")
	if !ok {
		t.Fatal("findTrigger did not find normalized trigger")
	}
	if enabled, _ := trigger["enabled"].(bool); !enabled {
		t.Fatalf("enabled = false, want true")
	}
}

func TestFindTriggerReturnsFalseForMissingType(t *testing.T) {
	if _, ok := findTrigger([]map[string]interface{}{{"type": "RISK_ASSESSMENT"}}, "TOO_MANY_LOGIN_FAILURES"); ok {
		t.Fatal("findTrigger returned true for missing trigger")
	}
}

func TestStringValuesSkipsNullUnknownAndPreservesValues(t *testing.T) {
	setValue, diags := types.SetValue(
		types.StringType,
		[]attr.Value{
			types.StringValue("notifier-1"),
			types.StringNull(),
			types.StringUnknown(),
			types.StringValue("notifier-2"),
		},
	)
	if diags.HasError() {
		t.Fatalf("build set value: %v", diags)
	}

	values := stringValues(setValue)

	if len(values) != 2 || values[0] != "notifier-1" || values[1] != "notifier-2" {
		t.Fatalf("values = %#v, want notifier-1,notifier-2", values)
	}
}

func TestStringValuesReturnsEmptyForNullOrUnknownSet(t *testing.T) {
	for _, setValue := range []types.Set{
		types.SetNull(types.StringType),
		types.SetUnknown(types.StringType),
	} {
		if values := stringValues(setValue); len(values) != 0 {
			t.Fatalf("values = %#v, want empty", values)
		}
	}
}

func TestAlertTriggerCreateAndUpdateRejectInvalidType(t *testing.T) {
	resourceUnderTest := &AlertTriggerResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := alertTriggerPlan(t, schemaResp.Schema, AlertTriggerModel{
		DomainID:         types.StringValue("domain-123"),
		Type:             types.StringValue("NOT_A_TRIGGER"),
		Enabled:          types.BoolValue(true),
		AlertNotifierIDs: stringSet(t, nil),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}
}

func TestReadIntoModelMapsAlertTriggerResponse(t *testing.T) {
	model := AlertTriggerModel{
		DomainID: types.StringValue("domain-id"),
		Type:     types.StringValue("risk_assessment"),
	}

	readIntoModel(&model, map[string]interface{}{
		"type":           "too_many_login_failures",
		"enabled":        true,
		"alertNotifiers": []interface{}{"notifier-1", "notifier-2"},
	})

	if model.ID.ValueString() != "domain-id/TOO_MANY_LOGIN_FAILURES" {
		t.Fatalf("id = %q, want domain-id/TOO_MANY_LOGIN_FAILURES", model.ID.ValueString())
	}
	if model.Type.ValueString() != "TOO_MANY_LOGIN_FAILURES" {
		t.Fatalf("type = %q, want TOO_MANY_LOGIN_FAILURES", model.Type.ValueString())
	}
	if !model.Enabled.ValueBool() {
		t.Fatal("enabled = false, want true")
	}
	values := stringValues(model.AlertNotifierIDs)
	if len(values) != 2 || values[0] != "notifier-1" || values[1] != "notifier-2" {
		t.Fatalf("alert notifier ids = %#v, want notifier-1,notifier-2", values)
	}
}

func TestReadIntoModelSetsEmptyNotifierSetWhenAPIOmitsValues(t *testing.T) {
	model := AlertTriggerModel{
		DomainID: types.StringValue("domain-id"),
		Type:     types.StringValue("risk_assessment"),
	}

	readIntoModel(&model, map[string]interface{}{
		"enabled": false,
	})

	if model.ID.ValueString() != "domain-id/RISK_ASSESSMENT" {
		t.Fatalf("id = %q, want domain-id/RISK_ASSESSMENT", model.ID.ValueString())
	}
	if model.Type.ValueString() != "RISK_ASSESSMENT" {
		t.Fatalf("type = %q, want RISK_ASSESSMENT", model.Type.ValueString())
	}
	if !model.AlertNotifierIDs.IsNull() && len(model.AlertNotifierIDs.Elements()) != 0 {
		t.Fatalf("alert notifier ids = %#v, want empty set", model.AlertNotifierIDs)
	}
}

func TestAlertTriggerCRUDPatchesOnlyManagedTrigger(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	var patchBodies [][]map[string]interface{}
	var methods []string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/triggers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			var body []map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode patch body: %v", err)
			}
			methods = append(methods, "patch")
			patchBodies = append(patchBodies, body)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"type":           body[0]["type"],
					"enabled":        body[0]["enabled"],
					"alertNotifiers": body[0]["alertNotifiers"],
				},
				{
					"type":           "RISK_ASSESSMENT",
					"enabled":        true,
					"alertNotifiers": []string{"other-notifier"},
				},
			})
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"type":           "TOO_MANY_LOGIN_FAILURES",
					"enabled":        true,
					"alertNotifiers": []string{"notifier-1", "notifier-2"},
				},
				{
					"type":           "RISK_ASSESSMENT",
					"enabled":        true,
					"alertNotifiers": []string{"other-notifier"},
				},
			})
		default:
			t.Fatalf("unexpected alert trigger method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &AlertTriggerResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := alertTriggerPlan(t, schemaResp.Schema, AlertTriggerModel{
		DomainID:         types.StringValue("domain-123"),
		Type:             types.StringValue("too_many_login_failures"),
		Enabled:          types.BoolValue(true),
		AlertNotifierIDs: stringSet(t, []string{"notifier-1", "notifier-2"}),
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

	updatePlan := alertTriggerPlan(t, schemaResp.Schema, AlertTriggerModel{
		DomainID:         types.StringValue("domain-123"),
		Type:             types.StringValue("TOO_MANY_LOGIN_FAILURES"),
		Enabled:          types.BoolValue(false),
		AlertNotifierIDs: stringSet(t, []string{"notifier-3"}),
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

	if got, want := len(patchBodies), 3; got != want {
		t.Fatalf("patch count = %d, want %d", got, want)
	}
	if got, want := patchBodies[0][0]["alertNotifiers"], []interface{}{"notifier-1", "notifier-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("create notifiers = %#v, want %#v", got, want)
	}
	if got, want := patchBodies[1][0]["alertNotifiers"], []interface{}{"notifier-3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("update notifiers = %#v, want %#v", got, want)
	}
	if got, want := patchBodies[2][0]["enabled"], false; got != want {
		t.Fatalf("delete enabled = %#v, want %#v", got, want)
	}
	if got, want := patchBodies[2][0]["alertNotifiers"], []interface{}{}; !reflect.DeepEqual(got, want) {
		t.Fatalf("delete notifiers = %#v, want empty %#v", got, want)
	}
	if want := []string{"patch", "read", "patch", "patch"}; !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
}

func TestAlertTriggerReadRemovesMissingTriggerAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []map[string]interface{}
		wantRemove bool
	}{
		{
			name:       "missing domain",
			statusCode: http.StatusNotFound,
			wantRemove: true,
		},
		{
			name:       "missing trigger",
			statusCode: http.StatusOK,
			body:       []map[string]interface{}{{"type": "RISK_ASSESSMENT", "enabled": true}},
			wantRemove: true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantRemove: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/triggers", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				if tt.statusCode != http.StatusOK {
					http.Error(w, "read failed", tt.statusCode)
					return
				}
				_ = json.NewEncoder(w).Encode(tt.body)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &AlertTriggerResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := alertTriggerState(t, schemaResp.Schema, AlertTriggerModel{
				ID:               types.StringValue("domain-123/TOO_MANY_LOGIN_FAILURES"),
				DomainID:         types.StringValue("domain-123"),
				Type:             types.StringValue("TOO_MANY_LOGIN_FAILURES"),
				Enabled:          types.BoolValue(true),
				AlertNotifierIDs: stringSet(t, []string{"notifier-1"}),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing alert trigger to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestAlertTriggerReportsLifecycleErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []map[string]interface{}
		run        func(*AlertTriggerResource, resourceschema.Schema)
	}{
		{
			name:       "create patch error",
			statusCode: http.StatusInternalServerError,
			run: func(resourceUnderTest *AlertTriggerResource, schema resourceschema.Schema) {
				plan := alertTriggerPlan(t, schema, AlertTriggerModel{
					DomainID:         types.StringValue("domain-123"),
					Type:             types.StringValue("TOO_MANY_LOGIN_FAILURES"),
					Enabled:          types.BoolValue(true),
					AlertNotifierIDs: stringSet(t, []string{"notifier-1"}),
				})
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected create diagnostics")
				}
			},
		},
		{
			name:       "update missing trigger in response",
			statusCode: http.StatusOK,
			body:       []map[string]interface{}{{"type": "RISK_ASSESSMENT", "enabled": true}},
			run: func(resourceUnderTest *AlertTriggerResource, schema resourceschema.Schema) {
				plan := alertTriggerPlan(t, schema, AlertTriggerModel{
					DomainID:         types.StringValue("domain-123"),
					Type:             types.StringValue("TOO_MANY_LOGIN_FAILURES"),
					Enabled:          types.BoolValue(false),
					AlertNotifierIDs: stringSet(t, []string{"notifier-2"}),
				})
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan}, resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected update diagnostics")
				}
			},
		},
		{
			name:       "delete patch error",
			statusCode: http.StatusInternalServerError,
			run: func(resourceUnderTest *AlertTriggerResource, schema resourceschema.Schema) {
				state := alertTriggerState(t, schema, AlertTriggerModel{
					ID:               types.StringValue("domain-123/TOO_MANY_LOGIN_FAILURES"),
					DomainID:         types.StringValue("domain-123"),
					Type:             types.StringValue("TOO_MANY_LOGIN_FAILURES"),
					Enabled:          types.BoolValue(true),
					AlertNotifierIDs: stringSet(t, []string{"notifier-1"}),
				})
				resp := &resource.DeleteResponse{}
				resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected delete diagnostics")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/triggers", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Fatalf("method = %s, want PATCH", r.Method)
				}
				if tt.statusCode != http.StatusOK {
					http.Error(w, "patch failed", tt.statusCode)
					return
				}
				_ = json.NewEncoder(w).Encode(tt.body)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &AlertTriggerResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			tt.run(resourceUnderTest, schemaResp.Schema)
		})
	}
}

func TestAlertTriggerDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/alerts/triggers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &AlertTriggerResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := alertTriggerState(t, schemaResp.Schema, AlertTriggerModel{
		ID:               types.StringValue("domain-123/TOO_MANY_LOGIN_FAILURES"),
		DomainID:         types.StringValue("domain-123"),
		Type:             types.StringValue("TOO_MANY_LOGIN_FAILURES"),
		Enabled:          types.BoolValue(true),
		AlertNotifierIDs: stringSet(t, []string{"notifier-1"}),
	})

	resp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", resp.Diagnostics)
	}
}

func TestAlertTriggerImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&AlertTriggerResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestAlertTriggerImportStateSetsDomainAndNormalizedType(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewAlertTriggerResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: alertTriggerState(t, schemaResp.Schema, AlertTriggerModel{
		ID:               types.StringValue("placeholder"),
		DomainID:         types.StringValue("placeholder"),
		Type:             types.StringValue("TOO_MANY_LOGIN_FAILURES"),
		Enabled:          types.BoolValue(true),
		AlertNotifierIDs: stringSet(t, nil),
	})}

	(&AlertTriggerResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/risk_assessment",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported AlertTriggerModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.ID.ValueString(), "domain-123/risk_assessment"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.Type.ValueString(), "RISK_ASSESSMENT"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
}

func alertTriggerPlan(t *testing.T, schema resourceschema.Schema, model AlertTriggerModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func alertTriggerState(t *testing.T, schema resourceschema.Schema, model AlertTriggerModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func stringSet(t *testing.T, values []string) types.Set {
	t.Helper()

	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	setValue, diags := types.SetValue(types.StringType, elements)
	if diags.HasError() {
		t.Fatalf("set value diagnostics: %#v", diags)
	}
	return setValue
}
