package role

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	NewRoleResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_role"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewRoleResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "description", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "assignable_type", false, true, false)
	assertListAttribute(t, resp.Schema.Attributes, "permissions", types.StringType)
	assertListAttribute(t, resp.Schema.Attributes, "oauth_scopes", types.StringType)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&RoleResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestBuildUpdateBodyClearsRemovedRoleLists(t *testing.T) {
	plan := RoleModel{
		Name: types.StringValue("updated"),
	}
	state := RoleModel{
		Name:        types.StringValue("updated"),
		Description: types.StringValue("old description"),
		Permissions: []types.String{
			types.StringValue("DOMAIN_READ"),
		},
		OAuthScopes: []types.String{
			types.StringValue("scope-id"),
		},
	}

	body := (&RoleResource{}).buildUpdateBody(plan, &state)

	if got := body["description"]; got != nil {
		t.Fatalf("description = %#v, want omitted when removed", got)
	}
	perms, ok := body["permissions"].([]string)
	if !ok {
		t.Fatalf("permissions = %#v, want []string", body["permissions"])
	}
	if len(perms) != 0 {
		t.Fatalf("permissions length = %d, want 0", len(perms))
	}
	scopes, ok := body["oauthScopes"].([]string)
	if !ok {
		t.Fatalf("oauthScopes = %#v, want []string", body["oauthScopes"])
	}
	if len(scopes) != 0 {
		t.Fatalf("oauthScopes length = %d, want 0", len(scopes))
	}
}

func TestReadIntoModelMapsRoleResponse(t *testing.T) {
	model := RoleModel{}

	(&RoleResource{}).readIntoModel(&model, map[string]interface{}{
		"id":             "role-id",
		"name":           "role-name",
		"description":    "description",
		"assignableType": "domain",
		"permissions":    []interface{}{"DOMAIN_READ"},
		"oauthScopes":    []interface{}{"scope-id"},
	})

	if model.ID.ValueString() != "role-id" {
		t.Fatalf("id = %q, want role-id", model.ID.ValueString())
	}
	if model.Name.ValueString() != "role-name" {
		t.Fatalf("name = %q, want role-name", model.Name.ValueString())
	}
	if model.Description.ValueString() != "description" {
		t.Fatalf("description = %q, want description", model.Description.ValueString())
	}
	if model.AssignableType.ValueString() != "DOMAIN" {
		t.Fatalf("assignableType = %q, want DOMAIN", model.AssignableType.ValueString())
	}
	if len(model.Permissions) != 1 || model.Permissions[0].ValueString() != "DOMAIN_READ" {
		t.Fatalf("permissions = %#v, want DOMAIN_READ", model.Permissions)
	}
	if len(model.OAuthScopes) != 1 || model.OAuthScopes[0].ValueString() != "scope-id" {
		t.Fatalf("oauthScopes = %#v, want scope-id", model.OAuthScopes)
	}
}

func TestRoleCRUDUsesCreateThenUpdateAndClearsManagedLists(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/roles", func(w http.ResponseWriter, r *http.Request) {
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
			"id":             "role-123",
			"name":           body["name"],
			"description":    body["description"],
			"assignableType": body["assignableType"],
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             "role-123",
				"name":           "role-name",
				"description":    "created",
				"assignableType": "domain",
				"permissions":    []interface{}{"DOMAIN_READ"},
				"oauthScopes":    []interface{}{"openid"},
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             "role-123",
				"name":           body["name"],
				"description":    body["description"],
				"assignableType": "domain",
				"permissions":    body["permissions"],
				"oauthScopes":    body["oauthScopes"],
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

	resourceUnderTest := &RoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := rolePlan(t, schemaResp.Schema, RoleModel{
		DomainID:       types.StringValue("domain-123"),
		Name:           types.StringValue("role-name"),
		Description:    types.StringValue("created"),
		AssignableType: types.StringValue("DOMAIN"),
		Permissions:    []types.String{types.StringValue("DOMAIN_READ")},
		OAuthScopes:    []types.String{types.StringValue("openid")},
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState RoleModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if got := stringSlice(createState.Permissions); !reflect.DeepEqual(got, []string{"DOMAIN_READ"}) {
		t.Fatalf("permissions after create = %#v", got)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}

	updatePlan := rolePlan(t, schemaResp.Schema, RoleModel{
		DomainID:       types.StringValue("domain-123"),
		Name:           types.StringValue("role-updated"),
		AssignableType: types.StringValue("DOMAIN"),
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

	if !reflect.DeepEqual(methods, []string{"create", "update", "read", "update", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if len(bodies) != 3 {
		t.Fatalf("bodies = %#v, want create, post-create update, update", bodies)
	}
	if _, ok := bodies[0]["permissions"]; ok {
		t.Fatalf("create body should not include permissions: %#v", bodies[0])
	}
	if _, ok := bodies[0]["oauthScopes"]; ok {
		t.Fatalf("create body should not include oauthScopes: %#v", bodies[0])
	}
	if got := bodies[1]["permissions"]; !reflect.DeepEqual(got, []interface{}{"DOMAIN_READ"}) {
		t.Fatalf("post-create permissions = %#v", got)
	}
	if got := bodies[1]["oauthScopes"]; !reflect.DeepEqual(got, []interface{}{"openid"}) {
		t.Fatalf("post-create oauthScopes = %#v", got)
	}
	if got := bodies[2]["permissions"]; !reflect.DeepEqual(got, []interface{}{}) {
		t.Fatalf("update permissions = %#v, want clear list", got)
	}
	if got := bodies[2]["oauthScopes"]; !reflect.DeepEqual(got, []interface{}{}) {
		t.Fatalf("update oauthScopes = %#v, want clear list", got)
	}
	if _, ok := bodies[2]["description"]; ok {
		t.Fatalf("update body should omit removed description: %#v", bodies[2])
	}
}

func TestRoleReadRemovesMissingRoleAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing role", statusCode: http.StatusNotFound, wantRemove: true},
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &RoleResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := roleState(t, schemaResp.Schema, RoleModel{
				ID:       types.StringValue("role-123"),
				DomainID: types.StringValue("domain-123"),
				Name:     types.StringValue("role-name"),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing role to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestRoleReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			http.Error(w, "update failed", http.StatusInternalServerError)
		case http.MethodDelete:
			http.Error(w, "delete failed", http.StatusInternalServerError)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &RoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := rolePlan(t, schemaResp.Schema, RoleModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("role-name"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := roleState(t, schemaResp.Schema, RoleModel{
		ID:       types.StringValue("role-123"),
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("role-name"),
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

func TestRoleCreateReportsPostCreateUpdateError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "role-123",
			"name": "role-name",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("item method = %s, want PUT", r.Method)
		}
		http.Error(w, "post-create update failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &RoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := rolePlan(t, schemaResp.Schema, RoleModel{
		DomainID:    types.StringValue("domain-123"),
		Name:        types.StringValue("role-name"),
		Permissions: []types.String{types.StringValue("DOMAIN_READ")},
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected post-create update diagnostics")
	}
}

func TestRoleImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&RoleResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestRoleConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &RoleResource{}
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

func TestRoleImportStateSetsDomainAndID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewRoleResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: roleState(t, schemaResp.Schema, RoleModel{
		ID:             types.StringValue("placeholder"),
		DomainID:       types.StringValue("placeholder"),
		Name:           types.StringValue("role-name"),
		AssignableType: types.StringValue("DOMAIN"),
	})}

	(&RoleResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/role-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported RoleModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.ID.ValueString(), "role-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func TestRoleCreateReadUpdateAndDeleteReportInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &RoleResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	stringListType := tftypes.List{ElementType: tftypes.String}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":              tftypes.String,
			"domain_id":       tftypes.Number,
			"name":            tftypes.String,
			"description":     tftypes.String,
			"assignable_type": tftypes.String,
			"permissions":     stringListType,
			"oauth_scopes":    stringListType,
		}},
		map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, "role-123"),
			"domain_id":       tftypes.NewValue(tftypes.Number, 123),
			"name":            tftypes.NewValue(tftypes.String, "role"),
			"description":     tftypes.NewValue(tftypes.String, "created"),
			"assignable_type": tftypes.NewValue(tftypes.String, "DOMAIN"),
			"permissions":     tftypes.NewValue(stringListType, []tftypes.Value{tftypes.NewValue(tftypes.String, "domain_user_read")}),
			"oauth_scopes":    tftypes.NewValue(stringListType, []tftypes.Value{tftypes.NewValue(tftypes.String, "openid")}),
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

func rolePlan(t *testing.T, schema resourceschema.Schema, model RoleModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func roleState(t *testing.T, schema resourceschema.Schema, model RoleModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func stringSlice(values []types.String) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.ValueString()
	}
	return result
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

func assertListAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, elemType attr.Type) {
	t.Helper()

	attr, ok := attrs[name].(schema.ListAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.ListAttribute", name, attrs[name])
	}
	if !attr.Optional || attr.Required || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want optional only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
	if !reflect.DeepEqual(attr.ElementType, elemType) {
		t.Fatalf("%s element type = %#v, want %#v", name, attr.ElementType, elemType)
	}
}
