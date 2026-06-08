package extensiongrant

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

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewExtensionGrantResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_extension_grant"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewExtensionGrantResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "type", "grant_type", "configuration"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"identity_provider", "create_user", "user_exists"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "create_user", "user_exists"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ExtensionGrantResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildUpdateBodyIncludesIdentityProviderWhenPlanned(t *testing.T) {
	t.Parallel()

	plan := ExtensionGrantModel{
		Name:             types.StringValue("jwt bearer"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{"issuer":"test"}`),
		IdentityProvider: types.StringValue("idp-1"),
		CreateUser:       types.BoolValue(true),
		UserExists:       types.BoolValue(false),
	}

	got := buildUpdateBody(plan, ExtensionGrantModel{})
	want := map[string]interface{}{
		"name":             "jwt bearer",
		"type":             "jwtbearer-am-extension-grant",
		"grantType":        "urn:ietf:params:oauth:grant-type:jwt-bearer",
		"configuration":    `{"issuer":"test"}`,
		"identityProvider": "idp-1",
		"createUser":       true,
		"userExists":       false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyClearsExistingIdentityProviderWhenPlanNull(t *testing.T) {
	t.Parallel()

	plan := ExtensionGrantModel{
		Name:             types.StringValue("jwt bearer"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{}`),
		IdentityProvider: types.StringNull(),
		CreateUser:       types.BoolValue(false),
		UserExists:       types.BoolValue(true),
	}
	state := ExtensionGrantModel{
		IdentityProvider: types.StringValue("idp-1"),
	}

	got := buildUpdateBody(plan, state)

	if got["identityProvider"] != "" {
		t.Fatalf("identityProvider = %#v, want clear string", got["identityProvider"])
	}
}

func TestBuildUpdateBodyOmitsIdentityProviderWhenPlanAndStateNull(t *testing.T) {
	t.Parallel()

	plan := ExtensionGrantModel{
		Name:             types.StringValue("jwt bearer"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{}`),
		IdentityProvider: types.StringNull(),
		CreateUser:       types.BoolValue(false),
		UserExists:       types.BoolValue(false),
	}

	got := buildUpdateBody(plan, ExtensionGrantModel{IdentityProvider: types.StringNull()})

	if _, ok := got["identityProvider"]; ok {
		t.Fatalf("identityProvider should be omitted: %#v", got)
	}
}

func TestReadIntoModelMapsExtensionGrantFields(t *testing.T) {
	t.Parallel()

	model := ExtensionGrantModel{}

	readIntoModel(&model, map[string]interface{}{
		"name":             "jwt bearer",
		"type":             "jwtbearer-am-extension-grant",
		"grantType":        "urn:ietf:params:oauth:grant-type:jwt-bearer",
		"configuration":    `{"issuer":"test"}`,
		"identityProvider": "idp-1",
		"createUser":       true,
		"userExists":       false,
	})

	if model.Name.ValueString() != "jwt bearer" ||
		model.Type.ValueString() != "jwtbearer-am-extension-grant" ||
		model.GrantType.ValueString() != "urn:ietf:params:oauth:grant-type:jwt-bearer" ||
		model.Configuration.ValueString() != `{"issuer":"test"}` ||
		model.IdentityProvider.ValueString() != "idp-1" ||
		!model.CreateUser.ValueBool() ||
		model.UserExists.ValueBool() {
		t.Fatalf("model = %#v", model)
	}
}

func TestReadIntoModelClearsEmptyIdentityProvider(t *testing.T) {
	t.Parallel()

	model := ExtensionGrantModel{
		IdentityProvider: types.StringValue("idp-1"),
	}

	readIntoModel(&model, map[string]interface{}{
		"identityProvider": "",
	})

	if !model.IdentityProvider.IsNull() {
		t.Fatalf("identityProvider = %#v, want null", model.IdentityProvider)
	}
}

func TestReadIntoModelKeepsNullIdentityProviderWhenMissing(t *testing.T) {
	t.Parallel()

	model := ExtensionGrantModel{
		IdentityProvider: types.StringNull(),
	}

	readIntoModel(&model, map[string]interface{}{})

	if !model.IdentityProvider.IsNull() {
		t.Fatalf("identityProvider = %#v, want null", model.IdentityProvider)
	}
}

func TestExtensionGrantCRUDClearsIdentityProvider(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/extensionGrants", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":               "grant-123",
			"name":             body["name"],
			"type":             body["type"],
			"grantType":        body["grantType"],
			"configuration":    body["configuration"],
			"identityProvider": body["identityProvider"],
			"createUser":       body["createUser"],
			"userExists":       body["userExists"],
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/extensionGrants/grant-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":               "grant-123",
				"name":             "jwt bearer",
				"type":             "jwtbearer-am-extension-grant",
				"grantType":        "urn:ietf:params:oauth:grant-type:jwt-bearer",
				"configuration":    `{"issuer":"test"}`,
				"identityProvider": "idp-1",
				"createUser":       true,
				"userExists":       false,
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":               "grant-123",
				"name":             body["name"],
				"type":             body["type"],
				"grantType":        body["grantType"],
				"configuration":    body["configuration"],
				"identityProvider": body["identityProvider"],
				"createUser":       body["createUser"],
				"userExists":       body["userExists"],
			})
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ExtensionGrantResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := extensionGrantPlan(t, schemaResp.Schema, ExtensionGrantModel{
		DomainID:         types.StringValue("domain-123"),
		Name:             types.StringValue("jwt bearer"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{"issuer":"test"}`),
		IdentityProvider: types.StringValue("idp-1"),
		CreateUser:       types.BoolValue(true),
		UserExists:       types.BoolValue(false),
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

	updatePlan := extensionGrantPlan(t, schemaResp.Schema, ExtensionGrantModel{
		DomainID:         types.StringValue("domain-123"),
		Name:             types.StringValue("jwt bearer updated"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{"issuer":"updated"}`),
		IdentityProvider: types.StringNull(),
		CreateUser:       types.BoolValue(false),
		UserExists:       types.BoolValue(true),
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

	if !reflect.DeepEqual(methods, []string{"create", "read", "update", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if got := bodies[0]["identityProvider"]; got != "idp-1" {
		t.Fatalf("create identityProvider = %#v, want idp-1", got)
	}
	if got := bodies[1]["identityProvider"]; got != "" {
		t.Fatalf("update identityProvider = %#v, want clear string", got)
	}
	if got := bodies[1]["createUser"]; got != false {
		t.Fatalf("update createUser = %#v, want false", got)
	}
	if got := bodies[1]["userExists"]; got != true {
		t.Fatalf("update userExists = %#v, want true", got)
	}
}

func TestExtensionGrantReadRemovesMissingGrantAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{
			name:       "missing grant",
			statusCode: http.StatusNotFound,
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/extensionGrants/grant-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &ExtensionGrantResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := extensionGrantState(t, schemaResp.Schema, ExtensionGrantModel{
				ID:               types.StringValue("grant-123"),
				DomainID:         types.StringValue("domain-123"),
				Name:             types.StringValue("jwt bearer"),
				Type:             types.StringValue("jwtbearer-am-extension-grant"),
				GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
				Configuration:    types.StringValue(`{"issuer":"test"}`),
				IdentityProvider: types.StringValue("idp-1"),
				CreateUser:       types.BoolValue(true),
				UserExists:       types.BoolValue(false),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing grant to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestExtensionGrantReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/extensionGrants", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/extensionGrants/grant-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &ExtensionGrantResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := extensionGrantPlan(t, schemaResp.Schema, ExtensionGrantModel{
		DomainID:         types.StringValue("domain-123"),
		Name:             types.StringValue("jwt bearer"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{"issuer":"test"}`),
		IdentityProvider: types.StringValue("idp-1"),
		CreateUser:       types.BoolValue(true),
		UserExists:       types.BoolValue(false),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := extensionGrantState(t, schemaResp.Schema, ExtensionGrantModel{
		ID:               types.StringValue("grant-123"),
		DomainID:         types.StringValue("domain-123"),
		Name:             types.StringValue("jwt bearer"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{"issuer":"test"}`),
		IdentityProvider: types.StringValue("idp-1"),
		CreateUser:       types.BoolValue(true),
		UserExists:       types.BoolValue(false),
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

func TestExtensionGrantImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&ExtensionGrantResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func extensionGrantPlan(t *testing.T, schema resourceschema.Schema, model ExtensionGrantModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func extensionGrantState(t *testing.T, schema resourceschema.Schema, model ExtensionGrantModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
