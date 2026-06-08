package deviceidentifier

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

func TestDeviceIdentifierMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewDeviceIdentifierResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_device_identifier" {
		t.Fatalf("type name = %q, want graviteeam_device_identifier", resp.TypeName)
	}
}

func TestDeviceIdentifierSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewDeviceIdentifierResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

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

func TestDeviceIdentifierConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &DeviceIdentifierResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestDeviceIdentifierConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&DeviceIdentifierResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		id                     string
		wantDomainID           string
		wantDeviceIdentifierID string
		wantOK                 bool
	}{
		{
			name:                   "valid",
			id:                     "domain-1/device-1",
			wantDomainID:           "domain-1",
			wantDeviceIdentifierID: "device-1",
			wantOK:                 true,
		},
		{
			name:                   "preserves splitN behavior",
			id:                     "domain-1/device-1/extra",
			wantDomainID:           "domain-1",
			wantDeviceIdentifierID: "device-1/extra",
			wantOK:                 true,
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

			gotDomainID, gotDeviceIdentifierID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotDeviceIdentifierID != tt.wantDeviceIdentifierID {
				t.Fatalf("device identifier ID = %q, want %q", gotDeviceIdentifierID, tt.wantDeviceIdentifierID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := DeviceIdentifierModel{
		Name:          types.StringValue("Fingerprint"),
		Type:          types.StringValue("fingerprintjs-v3-am-device-identifier"),
		Configuration: types.StringValue(`{"browserToken":"secret"}`),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":          "Fingerprint",
		"type":          "fingerprintjs-v3-am-device-identifier",
		"configuration": `{"browserToken":"secret"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := DeviceIdentifierModel{
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		Configuration: types.StringValue(`{"browserToken":"real-secret"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new-name",
		"type":          "fingerprintjs-v3-am-device-identifier",
		"configuration": `{"browserToken":"********"}`,
	})

	if got, want := model.Name.ValueString(), "new-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "fingerprintjs-v3-am-device-identifier"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"browserToken":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}
}

func TestDeviceIdentifierCRUDPreservesSecretConfiguration(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/device-identifiers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		body := decodeDeviceIdentifierBody(t, r)
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "device-123", "name": body["name"], "type": body["type"]})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/device-identifiers/device-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "device-123",
				"name":          "read-device",
				"type":          "fingerprintjs-v3-am-device-identifier",
				"configuration": `{"browserToken":"********"}`,
			})
		case http.MethodPut:
			body := decodeDeviceIdentifierBody(t, r)
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "device-123", "name": body["name"], "type": body["type"]})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &DeviceIdentifierResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := deviceIdentifierPlan(t, schemaResp.Schema, DeviceIdentifierModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("created-device"),
		Type:          types.StringValue("fingerprintjs-v3-am-device-identifier"),
		Configuration: types.StringValue(`{"browserToken":"real-secret"}`),
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
	var read DeviceIdentifierModel
	if diags := readResp.State.Get(context.Background(), &read); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if read.Configuration.ValueString() != `{"browserToken":"real-secret"}` {
		t.Fatalf("read configuration = %q", read.Configuration.ValueString())
	}

	updatePlan := deviceIdentifierPlan(t, schemaResp.Schema, DeviceIdentifierModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("updated-device"),
		Type:          types.StringValue("fingerprintjs-v3-am-device-identifier"),
		Configuration: types.StringValue(`{"browserToken":"updated-secret"}`),
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
		{"name": "created-device", "type": "fingerprintjs-v3-am-device-identifier", "configuration": `{"browserToken":"real-secret"}`},
		{"name": "updated-device", "type": "fingerprintjs-v3-am-device-identifier", "configuration": `{"browserToken":"updated-secret"}`},
	}
	if !reflect.DeepEqual(bodies, wantBodies) {
		t.Fatalf("bodies = %#v, want %#v", bodies, wantBodies)
	}
}

func TestDeviceIdentifierReadRemovesMissingIdentifierAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{"missing device identifier", http.StatusNotFound, true},
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/device-identifiers/device-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &DeviceIdentifierResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := deviceIdentifierState(t, schemaResp.Schema, DeviceIdentifierModel{
				ID:            types.StringValue("device-123"),
				DomainID:      types.StringValue("domain-123"),
				Name:          types.StringValue("Fingerprint"),
				Type:          types.StringValue("fingerprintjs-v3-am-device-identifier"),
				Configuration: types.StringValue(`{"browserToken":"real-secret"}`),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing device identifier to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestDeviceIdentifierReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/device-identifiers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/device-identifiers/device-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &DeviceIdentifierResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := deviceIdentifierPlan(t, schemaResp.Schema, DeviceIdentifierModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("Fingerprint"),
		Type:          types.StringValue("fingerprintjs-v3-am-device-identifier"),
		Configuration: types.StringValue(`{"browserToken":"real-secret"}`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := deviceIdentifierState(t, schemaResp.Schema, DeviceIdentifierModel{
		ID:            types.StringValue("device-123"),
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("Fingerprint"),
		Type:          types.StringValue("fingerprintjs-v3-am-device-identifier"),
		Configuration: types.StringValue(`{"browserToken":"real-secret"}`),
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

func TestDeviceIdentifierImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&DeviceIdentifierResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func decodeDeviceIdentifierBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func deviceIdentifierPlan(t *testing.T, schema resourceschema.Schema, model DeviceIdentifierModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func deviceIdentifierState(t *testing.T, schema resourceschema.Schema, model DeviceIdentifierModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
