package authorizationengine

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
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewAuthorizationEngineResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_authorization_engine"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewAuthorizationEngineResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "configuration", true, false, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&AuthorizationEngineResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		id           string
		wantDomainID string
		wantEngineID string
		wantOK       bool
	}{
		{
			name:         "valid",
			id:           "domain-1/engine-1",
			wantDomainID: "domain-1",
			wantEngineID: "engine-1",
			wantOK:       true,
		},
		{
			name:         "preserves splitN behavior",
			id:           "domain-1/engine-1/extra",
			wantDomainID: "domain-1",
			wantEngineID: "engine-1/extra",
			wantOK:       true,
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

			gotDomainID, gotEngineID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotEngineID != tt.wantEngineID {
				t.Fatalf("engine ID = %q, want %q", gotEngineID, tt.wantEngineID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := AuthorizationEngineModel{
		Name:          types.StringValue("OpenFGA"),
		Type:          types.StringValue("openfga"),
		Configuration: types.StringValue(`{"storeId":"store-1"}`),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":          "OpenFGA",
		"type":          "openfga",
		"configuration": `{"storeId":"store-1"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	model := AuthorizationEngineModel{
		ID:            types.StringValue("old-id"),
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		Configuration: types.StringValue(`{"storeId":"old"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"id":            "engine-1",
		"name":          "OpenFGA",
		"type":          "openfga",
		"configuration": `{"storeId":"new"}`,
	})

	if got, want := model.ID.ValueString(), "engine-1"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.Name.ValueString(), "OpenFGA"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "openfga"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"storeId":"new"}`; got != want {
		t.Fatalf("configuration = %q, want %q", got, want)
	}
}

func TestAuthorizationEngineCRUDRoundTripsConfiguration(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/authorization-engines", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected authorization engine collection method %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		methods = append(methods, "create")
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            "engine-123",
			"name":          body["name"],
			"type":          body["type"],
			"configuration": body["configuration"],
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/authorization-engines/engine-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "engine-123",
				"name":          "OpenFGA",
				"type":          "openfga",
				"configuration": `{"storeId":"store-1"}`,
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "engine-123",
				"name":          body["name"],
				"type":          body["type"],
				"configuration": body["configuration"],
			})
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected authorization engine item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &AuthorizationEngineResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := authorizationEnginePlan(t, schemaResp.Schema, AuthorizationEngineModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("OpenFGA"),
		Type:          types.StringValue("openfga"),
		Configuration: types.StringValue(`{"storeId":"store-1"}`),
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

	updatePlan := authorizationEnginePlan(t, schemaResp.Schema, AuthorizationEngineModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("OpenFGA updated"),
		Type:          types.StringValue("openfga"),
		Configuration: types.StringValue(`{"storeId":"store-2"}`),
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

	if got, want := bodies[1]["configuration"], `{"storeId":"store-2"}`; got != want {
		t.Fatalf("update configuration = %#v, want %#v", got, want)
	}
	if want := []string{"create", "read", "update", "delete"}; !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
}

func TestAuthorizationEngineReadRemovesMissingEngineAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing engine", statusCode: http.StatusNotFound, wantRemove: true},
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/authorization-engines/engine-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &AuthorizationEngineResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := authorizationEngineState(t, schemaResp.Schema, AuthorizationEngineModel{
				ID:            types.StringValue("engine-123"),
				DomainID:      types.StringValue("domain-123"),
				Name:          types.StringValue("OpenFGA"),
				Type:          types.StringValue("openfga"),
				Configuration: types.StringValue(`{"storeId":"store-1"}`),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing authorization engine to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestAuthorizationEngineReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/authorization-engines", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/authorization-engines/engine-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &AuthorizationEngineResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := authorizationEnginePlan(t, schemaResp.Schema, AuthorizationEngineModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("OpenFGA"),
		Type:          types.StringValue("openfga"),
		Configuration: types.StringValue(`{"storeId":"store-1"}`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := authorizationEngineState(t, schemaResp.Schema, AuthorizationEngineModel{
		ID:            types.StringValue("engine-123"),
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("OpenFGA"),
		Type:          types.StringValue("openfga"),
		Configuration: types.StringValue(`{"storeId":"store-1"}`),
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

func TestAuthorizationEngineImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&AuthorizationEngineResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestAuthorizationEngineConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &AuthorizationEngineResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if resourceUnderTest.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestAuthorizationEngineImportStateSetsDomainAndID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewAuthorizationEngineResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: authorizationEngineState(t, schemaResp.Schema, AuthorizationEngineModel{
		ID:            types.StringValue("placeholder"),
		DomainID:      types.StringValue("placeholder"),
		Name:          types.StringValue("OpenFGA"),
		Type:          types.StringValue("openfga"),
		Configuration: types.StringValue(`{"storeId":"store-1"}`),
	})}

	(&AuthorizationEngineResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/engine-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported AuthorizationEngineModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.ID.ValueString(), "engine-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func TestAuthorizationEngineCreateReadUpdateAndDeleteReportInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &AuthorizationEngineResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"domain_id":     tftypes.Number,
			"name":          tftypes.String,
			"type":          tftypes.String,
			"configuration": tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":            tftypes.NewValue(tftypes.String, "engine-123"),
			"domain_id":     tftypes.NewValue(tftypes.Number, 123),
			"name":          tftypes.NewValue(tftypes.String, "OpenFGA"),
			"type":          tftypes.NewValue(tftypes.String, "openfga"),
			"configuration": tftypes.NewValue(tftypes.String, "{}"),
		},
	)

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw}}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw}}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw}}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func authorizationEnginePlan(t *testing.T, schema resourceschema.Schema, model AuthorizationEngineModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func authorizationEngineState(t *testing.T, schema resourceschema.Schema, model AuthorizationEngineModel) tfsdk.State {
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
