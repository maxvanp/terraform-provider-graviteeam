package botdetection

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewBotDetectionResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_bot_detection"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewBotDetectionResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "detection_type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "configuration", true, false, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&BotDetectionResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		id                 string
		wantDomainID       string
		wantBotDetectionID string
		wantOK             bool
	}{
		{
			name:               "valid",
			id:                 "domain-1/bot-1",
			wantDomainID:       "domain-1",
			wantBotDetectionID: "bot-1",
			wantOK:             true,
		},
		{
			name:               "preserves splitN behavior",
			id:                 "domain-1/bot-1/extra",
			wantDomainID:       "domain-1",
			wantBotDetectionID: "bot-1/extra",
			wantOK:             true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotBotDetectionID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotBotDetectionID != tt.wantBotDetectionID {
				t.Fatalf("bot detection ID = %q, want %q", gotBotDetectionID, tt.wantBotDetectionID)
			}
		})
	}
}

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := BotDetectionModel{
		Name:          types.StringValue("reCAPTCHA"),
		Type:          types.StringValue("google-recaptcha-v3-am-bot-detection"),
		DetectionType: types.StringValue("CAPTCHA"),
		Configuration: types.StringValue(`{"siteKey":"site","secretKey":"secret"}`),
	}

	got := buildCreateBody(plan)
	want := map[string]interface{}{
		"name":          "reCAPTCHA",
		"type":          "google-recaptcha-v3-am-bot-detection",
		"detectionType": "CAPTCHA",
		"configuration": `{"siteKey":"site","secretKey":"secret"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("create body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyOmitsReplaceOnlyDetectionType(t *testing.T) {
	t.Parallel()

	plan := BotDetectionModel{
		Name:          types.StringValue("reCAPTCHA"),
		Type:          types.StringValue("google-recaptcha-v3-am-bot-detection"),
		DetectionType: types.StringValue("CAPTCHA"),
		Configuration: types.StringValue(`{"siteKey":"site","secretKey":"secret"}`),
	}

	got := buildUpdateBody(plan)
	want := map[string]interface{}{
		"name":          "reCAPTCHA",
		"type":          "google-recaptcha-v3-am-bot-detection",
		"configuration": `{"siteKey":"site","secretKey":"secret"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("update body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := BotDetectionModel{
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		DetectionType: types.StringValue("old-detection"),
		Configuration: types.StringValue(`{"secretKey":"real-secret"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new-name",
		"type":          "new-type",
		"detectionType": "CAPTCHA",
		"configuration": `{"secretKey":"********"}`,
	})

	if got, want := model.Name.ValueString(), "new-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "new-type"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.DetectionType.ValueString(), "CAPTCHA"; got != want {
		t.Fatalf("detection type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"secretKey":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}
}

func TestBotDetectionCRUDPreservesPlannedConfiguration(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/bot-detections", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected bot detection collection method %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		methods = append(methods, "create")
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "bot-123"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/bot-detections/bot-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "bot-123",
				"name":          "reCAPTCHA",
				"type":          "google-recaptcha-v3-am-bot-detection",
				"detectionType": "CAPTCHA",
				"configuration": `{"secretKey":"********"}`,
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "bot-123"})
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected bot detection item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &BotDetectionResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := botDetectionPlan(t, schemaResp.Schema, BotDetectionModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("reCAPTCHA"),
		Type:          types.StringValue("google-recaptcha-v3-am-bot-detection"),
		DetectionType: types.StringValue("CAPTCHA"),
		Configuration: types.StringValue(`{"secretKey":"real-secret"}`),
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
	var readState BotDetectionModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := readState.Configuration.ValueString(), `{"secretKey":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}

	updatePlan := botDetectionPlan(t, schemaResp.Schema, BotDetectionModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("reCAPTCHA updated"),
		Type:          types.StringValue("google-recaptcha-v3-am-bot-detection"),
		DetectionType: types.StringValue("CAPTCHA"),
		Configuration: types.StringValue(`{"secretKey":"new-secret"}`),
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

	if got, want := bodies[1]["configuration"], `{"secretKey":"new-secret"}`; got != want {
		t.Fatalf("update configuration = %#v, want %#v", got, want)
	}
	if _, ok := bodies[1]["detectionType"]; ok {
		t.Fatalf("update body should omit detectionType: %#v", bodies[1])
	}
	if want := []string{"create", "read", "update", "delete"}; !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
}

func TestBotDetectionReadRemovesMissingBotDetectionAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing bot detection", statusCode: http.StatusNotFound, wantRemove: true},
		{name: "server error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/bot-detections/bot-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &BotDetectionResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := botDetectionState(t, schemaResp.Schema, BotDetectionModel{
				ID:            types.StringValue("bot-123"),
				DomainID:      types.StringValue("domain-123"),
				Name:          types.StringValue("reCAPTCHA"),
				Type:          types.StringValue("google-recaptcha-v3-am-bot-detection"),
				DetectionType: types.StringValue("CAPTCHA"),
				Configuration: types.StringValue(`{"secretKey":"real-secret"}`),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing bot detection to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestBotDetectionReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/bot-detections", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/bot-detections/bot-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			http.Error(w, "update failed", http.StatusInternalServerError)
		case http.MethodDelete:
			http.Error(w, "delete failed", http.StatusInternalServerError)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &BotDetectionResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := botDetectionPlan(t, schemaResp.Schema, BotDetectionModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("reCAPTCHA"),
		Type:          types.StringValue("google-recaptcha-v3-am-bot-detection"),
		DetectionType: types.StringValue("CAPTCHA"),
		Configuration: types.StringValue(`{"secretKey":"real-secret"}`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := botDetectionState(t, schemaResp.Schema, BotDetectionModel{
		ID:            types.StringValue("bot-123"),
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("reCAPTCHA"),
		Type:          types.StringValue("google-recaptcha-v3-am-bot-detection"),
		DetectionType: types.StringValue("CAPTCHA"),
		Configuration: types.StringValue(`{"secretKey":"real-secret"}`),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestBotDetectionImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&BotDetectionResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func botDetectionPlan(t *testing.T, schema resourceschema.Schema, model BotDetectionModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func botDetectionState(t *testing.T, schema resourceschema.Schema, model BotDetectionModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
