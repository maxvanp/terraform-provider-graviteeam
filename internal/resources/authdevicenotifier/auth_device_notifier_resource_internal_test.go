package authdevicenotifier

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

func TestAuthDeviceNotifierMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewAuthDeviceNotifierResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_auth_device_notifier" {
		t.Fatalf("type name = %q, want graviteeam_auth_device_notifier", resp.TypeName)
	}
}

func TestAuthDeviceNotifierSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewAuthDeviceNotifierResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "type", "configuration"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
}

func TestAuthDeviceNotifierConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &AuthDeviceNotifierResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestAuthDeviceNotifierConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&AuthDeviceNotifierResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		id             string
		wantDomainID   string
		wantNotifierID string
		wantOK         bool
	}{
		{
			name:           "valid",
			id:             "domain-1/notifier-1",
			wantDomainID:   "domain-1",
			wantNotifierID: "notifier-1",
			wantOK:         true,
		},
		{
			name:           "preserves splitN behavior",
			id:             "domain-1/notifier-1/extra",
			wantDomainID:   "domain-1",
			wantNotifierID: "notifier-1/extra",
			wantOK:         true,
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

			gotDomainID, gotNotifierID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotNotifierID != tt.wantNotifierID {
				t.Fatalf("notifier ID = %q, want %q", gotNotifierID, tt.wantNotifierID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := AuthDeviceNotifierModel{
		Name:          types.StringValue("HTTP Notifier"),
		Type:          types.StringValue("http-am-authdevice-notifier"),
		Configuration: types.StringValue(`{"endpoint":"https://example.com","headerValue":"secret"}`),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":          "HTTP Notifier",
		"type":          "http-am-authdevice-notifier",
		"configuration": `{"endpoint":"https://example.com","headerValue":"secret"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := AuthDeviceNotifierModel{
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		Configuration: types.StringValue(`{"headerValue":"real-secret"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new-name",
		"type":          "http-am-authdevice-notifier",
		"configuration": `{"headerValue":"********"}`,
	})

	if got, want := model.Name.ValueString(), "new-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "http-am-authdevice-notifier"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"headerValue":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}
}

func TestAuthDeviceNotifierCRUDPreservesSecretConfiguration(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/auth-device-notifiers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		body := decodeAuthDeviceNotifierBody(t, r)
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "notifier-123", "name": body["name"], "type": body["type"]})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/auth-device-notifiers/notifier-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "notifier-123",
				"name":          "read-notifier",
				"type":          "http-am-authdevice-notifier",
				"configuration": `{"headerValue":"********"}`,
			})
		case http.MethodPut:
			body := decodeAuthDeviceNotifierBody(t, r)
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "notifier-123", "name": body["name"], "type": body["type"]})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &AuthDeviceNotifierResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := authDeviceNotifierPlan(t, schemaResp.Schema, AuthDeviceNotifierModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("created-notifier"),
		Type:          types.StringValue("http-am-authdevice-notifier"),
		Configuration: types.StringValue(`{"headerValue":"real-secret"}`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var created AuthDeviceNotifierModel
	if diags := createResp.State.Get(context.Background(), &created); diags.HasError() {
		t.Fatalf("get created state: %#v", diags)
	}
	if created.ID.ValueString() != "notifier-123" {
		t.Fatalf("created id = %q", created.ID.ValueString())
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var read AuthDeviceNotifierModel
	if diags := readResp.State.Get(context.Background(), &read); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if read.Configuration.ValueString() != `{"headerValue":"real-secret"}` {
		t.Fatalf("read configuration = %q", read.Configuration.ValueString())
	}

	updatePlan := authDeviceNotifierPlan(t, schemaResp.Schema, AuthDeviceNotifierModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("updated-notifier"),
		Type:          types.StringValue("http-am-authdevice-notifier"),
		Configuration: types.StringValue(`{"headerValue":"updated-secret"}`),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: updatePlan, State: readResp.State}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}

	deleteResp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
	wantBodies := []map[string]interface{}{
		{"name": "created-notifier", "type": "http-am-authdevice-notifier", "configuration": `{"headerValue":"real-secret"}`},
		{"name": "updated-notifier", "type": "http-am-authdevice-notifier", "configuration": `{"headerValue":"updated-secret"}`},
	}
	if !reflect.DeepEqual(bodies, wantBodies) {
		t.Fatalf("bodies = %#v, want %#v", bodies, wantBodies)
	}
}

func TestAuthDeviceNotifierReadRemovesMissingNotifierAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{"missing auth device notifier", http.StatusNotFound, true},
		{"server error", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/auth-device-notifiers/notifier-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &AuthDeviceNotifierResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := authDeviceNotifierState(t, schemaResp.Schema, AuthDeviceNotifierModel{
				ID:            types.StringValue("notifier-123"),
				DomainID:      types.StringValue("domain-123"),
				Name:          types.StringValue("HTTP Notifier"),
				Type:          types.StringValue("http-am-authdevice-notifier"),
				Configuration: types.StringValue(`{"headerValue":"real-secret"}`),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing auth device notifier to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestAuthDeviceNotifierReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/auth-device-notifiers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/auth-device-notifiers/notifier-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &AuthDeviceNotifierResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := authDeviceNotifierPlan(t, schemaResp.Schema, AuthDeviceNotifierModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("HTTP Notifier"),
		Type:          types.StringValue("http-am-authdevice-notifier"),
		Configuration: types.StringValue(`{"headerValue":"real-secret"}`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := authDeviceNotifierState(t, schemaResp.Schema, AuthDeviceNotifierModel{
		ID:            types.StringValue("notifier-123"),
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("HTTP Notifier"),
		Type:          types.StringValue("http-am-authdevice-notifier"),
		Configuration: types.StringValue(`{"headerValue":"real-secret"}`),
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

func TestAuthDeviceNotifierImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&AuthDeviceNotifierResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func decodeAuthDeviceNotifierBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func authDeviceNotifierPlan(t *testing.T, schema resourceschema.Schema, model AuthDeviceNotifierModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func authDeviceNotifierState(t *testing.T, schema resourceschema.Schema, model AuthDeviceNotifierModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
