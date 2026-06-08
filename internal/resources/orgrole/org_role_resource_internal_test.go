package orgrole

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
	NewOrgRoleResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_role"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgRoleResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "description", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "assignable_type", true, false, false)
	assertListAttribute(t, resp.Schema.Attributes, "permissions", types.StringType)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgRoleResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgRoleResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgRoleResource{}
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

func TestBuildUpdateBodyAppliesPlannedFieldsAndPermissions(t *testing.T) {
	t.Parallel()

	plan := OrgRoleModel{
		Name:        types.StringValue("platform admin"),
		Description: types.StringValue("Admin role"),
		Permissions: []types.String{
			types.StringValue("organization_role_read"),
			types.StringValue("organization_role_update"),
		},
	}

	got := buildUpdateBody(plan, OrgRoleModel{})
	want := map[string]interface{}{
		"name":        "platform admin",
		"description": "Admin role",
		"permissions": []string{"organization_role_read", "organization_role_update"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyClearsDescriptionAndPermissions(t *testing.T) {
	t.Parallel()

	plan := OrgRoleModel{
		Name:        types.StringValue("platform admin"),
		Description: types.StringNull(),
		Permissions: nil,
	}
	state := OrgRoleModel{
		Description: types.StringValue("old"),
		Permissions: []types.String{
			types.StringValue("organization_role_read"),
		},
	}

	got := buildUpdateBody(plan, state)

	if got["description"] != "" {
		t.Fatalf("description = %#v, want empty string", got["description"])
	}
	if !reflect.DeepEqual(got["permissions"], []string{}) {
		t.Fatalf("permissions = %#v, want clear list", got["permissions"])
	}
}

func TestReadIntoModelMapsOrganizationRoleFields(t *testing.T) {
	t.Parallel()

	model := OrgRoleModel{}

	readIntoModel(&model, map[string]interface{}{
		"name":           "platform admin",
		"description":    "Admin role",
		"assignableType": "organization",
		"permissions": []interface{}{
			"organization_role_read",
			"organization_role_update",
		},
	})

	if model.Name.ValueString() != "platform admin" ||
		model.Description.ValueString() != "Admin role" ||
		model.AssignableType.ValueString() != "ORGANIZATION" {
		t.Fatalf("model fields not mapped: %#v", model)
	}
	if got := []string{model.Permissions[0].ValueString(), model.Permissions[1].ValueString()}; !reflect.DeepEqual(got, []string{"organization_role_read", "organization_role_update"}) {
		t.Fatalf("permissions = %#v", got)
	}
}

func TestReadIntoModelClearsEmptyOrganizationRoleOptionals(t *testing.T) {
	t.Parallel()

	model := OrgRoleModel{
		Description: types.StringValue("old"),
		Permissions: []types.String{
			types.StringValue("organization_role_read"),
		},
	}

	readIntoModel(&model, map[string]interface{}{
		"description": "",
		"permissions": []interface{}{},
	})

	if !model.Description.IsNull() {
		t.Fatalf("description = %#v, want null", model.Description)
	}
	if model.Permissions != nil {
		t.Fatalf("permissions = %#v, want nil", model.Permissions)
	}
}

func TestOrgRoleCRUDUsesPostCreateReadAndClearsManagedFields(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/roles", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             "role-123",
				"name":           "org-role",
				"description":    "created",
				"assignableType": "organization",
				"permissions":    []interface{}{"organization_role_read"},
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
				"assignableType": "organization",
				"permissions":    body["permissions"],
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

	resourceUnderTest := &OrgRoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgRolePlan(t, schemaResp.Schema, OrgRoleModel{
		Name:           types.StringValue("org-role"),
		Description:    types.StringValue("created"),
		AssignableType: types.StringValue("ORGANIZATION"),
		Permissions:    []types.String{types.StringValue("organization_role_read")},
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState OrgRoleModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if got := orgRoleStringSlice(createState.Permissions); !reflect.DeepEqual(got, []string{"organization_role_read"}) {
		t.Fatalf("permissions after create = %#v", got)
	}

	updatePlan := orgRolePlan(t, schemaResp.Schema, OrgRoleModel{
		Name:           types.StringValue("org-role-updated"),
		Description:    types.StringNull(),
		AssignableType: types.StringValue("ORGANIZATION"),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: createResp.State,
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
	if got := bodies[1]["permissions"]; !reflect.DeepEqual(got, []interface{}{"organization_role_read"}) {
		t.Fatalf("post-create permissions = %#v", got)
	}
	if got := bodies[2]["description"]; got != "" {
		t.Fatalf("update description = %#v, want clear string", got)
	}
	if got := bodies[2]["permissions"]; !reflect.DeepEqual(got, []interface{}{}) {
		t.Fatalf("update permissions = %#v, want clear list", got)
	}
}

func TestOrgRoleReadRemovesMissingRoleAndReportsErrors(t *testing.T) {
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
			mux.HandleFunc("/management/organizations/DEFAULT/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &OrgRoleResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := orgRoleState(t, schemaResp.Schema, OrgRoleModel{
				ID:             types.StringValue("role-123"),
				Name:           types.StringValue("org-role"),
				Description:    types.StringValue("Role"),
				AssignableType: types.StringValue("ORGANIZATION"),
				Permissions:    []types.String{types.StringValue("organization_role_read")},
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

func TestOrgRoleReadMapsRemoteRole(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             "role-123",
			"name":           "platform admin",
			"description":    "Admin role",
			"assignableType": "organization",
			"permissions": []interface{}{
				"organization_role_read",
				"organization_role_update",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgRoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := orgRoleState(t, schemaResp.Schema, OrgRoleModel{
		ID:             types.StringValue("role-123"),
		Name:           types.StringValue("old"),
		Description:    types.StringValue("old"),
		AssignableType: types.StringValue("ORGANIZATION"),
		Permissions:    []types.String{types.StringValue("organization_role_read")},
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState OrgRoleModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := readState.Name.ValueString(), "platform admin"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got := []string{readState.Permissions[0].ValueString(), readState.Permissions[1].ValueString()}; !reflect.DeepEqual(got, []string{"organization_role_read", "organization_role_update"}) {
		t.Fatalf("permissions = %#v", got)
	}
}

func TestOrgRoleReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &OrgRoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := orgRolePlan(t, schemaResp.Schema, OrgRoleModel{
		Name:           types.StringValue("org-role"),
		Description:    types.StringValue("Role"),
		AssignableType: types.StringValue("ORGANIZATION"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := orgRoleState(t, schemaResp.Schema, OrgRoleModel{
		ID:             types.StringValue("role-123"),
		Name:           types.StringValue("org-role"),
		Description:    types.StringValue("Role"),
		AssignableType: types.StringValue("ORGANIZATION"),
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

func TestOrgRoleCreateReportsPostCreateUpdateError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             "role-123",
			"name":           "org-role",
			"assignableType": "ORGANIZATION",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		http.Error(w, "post-create update failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgRoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := orgRolePlan(t, schemaResp.Schema, OrgRoleModel{
		Name:           types.StringValue("org-role"),
		AssignableType: types.StringValue("ORGANIZATION"),
		Permissions:    []types.String{types.StringValue("organization_role_read")},
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected post-create update diagnostics")
	}
}

func TestOrgRoleCreateReportsPostCreateReadError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             "role-123",
			"name":           "org-role",
			"assignableType": "ORGANIZATION",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "role-123"})
		case http.MethodGet:
			http.Error(w, "post-create read failed", http.StatusInternalServerError)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgRoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := orgRolePlan(t, schemaResp.Schema, OrgRoleModel{
		Name:           types.StringValue("org-role"),
		AssignableType: types.StringValue("ORGANIZATION"),
		Permissions:    []types.String{types.StringValue("organization_role_read")},
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected post-create read diagnostics")
	}
}

func TestOrgRoleDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/roles/role-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgRoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := orgRoleState(t, schemaResp.Schema, OrgRoleModel{
		ID:             types.StringValue("role-123"),
		Name:           types.StringValue("org-role"),
		Description:    types.StringValue("Role"),
		AssignableType: types.StringValue("ORGANIZATION"),
	})

	resp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", resp.Diagnostics)
	}
}

func TestOrgRoleImportStateSetsID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewOrgRoleResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: orgRoleState(t, schemaResp.Schema, OrgRoleModel{
		ID:             types.StringValue("placeholder"),
		Name:           types.StringValue("org-role"),
		Description:    types.StringValue("Role"),
		AssignableType: types.StringValue("ORGANIZATION"),
	})}

	(&OrgRoleResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "role-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported OrgRoleModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.ID.ValueString(), "role-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func TestOrgRoleCreateReadUpdateAndDeleteReportInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgRoleResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	permissionsType := tftypes.List{ElementType: tftypes.String}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":              tftypes.String,
			"name":            tftypes.Number,
			"description":     tftypes.String,
			"assignable_type": tftypes.String,
			"permissions":     permissionsType,
		}},
		map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, "role-123"),
			"name":            tftypes.NewValue(tftypes.Number, 123),
			"description":     tftypes.NewValue(tftypes.String, "created"),
			"assignable_type": tftypes.NewValue(tftypes.String, "ORGANIZATION"),
			"permissions":     tftypes.NewValue(permissionsType, []tftypes.Value{tftypes.NewValue(tftypes.String, "organization_member_read")}),
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

func orgRolePlan(t *testing.T, schema resourceschema.Schema, model OrgRoleModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func orgRoleState(t *testing.T, schema resourceschema.Schema, model OrgRoleModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func orgRoleStringSlice(values []types.String) []string {
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
