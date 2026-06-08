package orggroup

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
	NewOrgGroupResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_group"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgGroupResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["name"]; attr == nil || !attr.IsRequired() {
		t.Fatalf("name should be required")
	}
	for _, name := range []string{"description", "members", "roles"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; attr == nil || !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgGroupResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgGroupResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestBuildBodyAppliesPlannedOrganizationGroupCollections(t *testing.T) {
	t.Parallel()

	model := OrgGroupModel{
		Name:        types.StringValue("admins"),
		Description: types.StringValue("Admin group"),
		Members:     []types.String{types.StringValue("user-1"), types.StringValue("user-2")},
		Roles:       []types.String{types.StringValue("role-1")},
	}

	got := buildBody(model, OrgGroupModel{})
	want := map[string]interface{}{
		"name":        "admins",
		"description": "Admin group",
		"members":     []string{"user-1", "user-2"},
		"roles":       []string{"role-1"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyClearsDescriptionMembersAndRoles(t *testing.T) {
	t.Parallel()

	model := OrgGroupModel{
		Name:        types.StringValue("admins"),
		Description: types.StringNull(),
		Members:     nil,
		Roles:       nil,
	}
	state := OrgGroupModel{
		Description: types.StringValue("old"),
		Members:     []types.String{types.StringValue("user-1")},
		Roles:       []types.String{types.StringValue("role-1")},
	}

	got := buildBody(model, state)

	if got["description"] != "" {
		t.Fatalf("description = %#v, want empty string", got["description"])
	}
	if !reflect.DeepEqual(got["members"], []string{}) {
		t.Fatalf("members = %#v, want clear list", got["members"])
	}
	if !reflect.DeepEqual(got["roles"], []string{}) {
		t.Fatalf("roles = %#v, want clear list", got["roles"])
	}
}

func TestReadIntoModelMapsOrganizationGroupFields(t *testing.T) {
	t.Parallel()

	model := OrgGroupModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":          "group-1",
		"name":        "admins",
		"description": "Admin group",
		"members":     []interface{}{"user-1", "user-2"},
		"roles":       []interface{}{"role-1"},
	})

	if model.ID.ValueString() != "group-1" ||
		model.Name.ValueString() != "admins" ||
		model.Description.ValueString() != "Admin group" {
		t.Fatalf("model fields not mapped: %#v", model)
	}
	if got := stringValues(model.Members); !reflect.DeepEqual(got, []string{"user-1", "user-2"}) {
		t.Fatalf("members = %#v", got)
	}
	if got := stringValues(model.Roles); !reflect.DeepEqual(got, []string{"role-1"}) {
		t.Fatalf("roles = %#v", got)
	}
}

func TestReadIntoModelClearsEmptyOrganizationGroupCollections(t *testing.T) {
	t.Parallel()

	model := OrgGroupModel{
		Description: types.StringValue("old"),
		Members:     []types.String{types.StringValue("user-1")},
		Roles:       []types.String{types.StringValue("role-1")},
	}

	readIntoModel(&model, map[string]interface{}{
		"description": "",
		"members":     []interface{}{},
		"roles":       []interface{}{},
	})

	if !model.Description.IsNull() {
		t.Fatalf("description = %#v, want null", model.Description)
	}
	if model.Members != nil || model.Roles != nil {
		t.Fatalf("collections = members %#v roles %#v, want nil", model.Members, model.Roles)
	}
}

func TestInterfaceStringsIgnoresNonStringValuesAsZeroValue(t *testing.T) {
	t.Parallel()

	got := interfaceStrings([]interface{}{"user-1", 42})

	if got[0].ValueString() != "user-1" || !got[1].IsNull() {
		t.Fatalf("converted values = %#v", got)
	}
}

func TestOrgGroupCRUDPreservesManagedCollections(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups", func(w http.ResponseWriter, r *http.Request) {
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
			"id":          "group-123",
			"name":        body["name"],
			"description": body["description"],
			"members":     body["members"],
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "group-123",
				"name":        "admins",
				"description": "Admin group",
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "group-123",
				"name":        body["name"],
				"description": body["description"],
				"members":     body["members"],
				"roles":       body["roles"],
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

	resourceUnderTest := &OrgGroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgGroupPlan(t, schemaResp.Schema, OrgGroupModel{
		Name:        types.StringValue("admins"),
		Description: types.StringValue("Admin group"),
		Members:     []types.String{types.StringValue("user-1")},
		Roles:       []types.String{types.StringValue("role-1")},
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
	var readState OrgGroupModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got := stringValues(readState.Members); !reflect.DeepEqual(got, []string{"user-1"}) {
		t.Fatalf("read members = %#v", got)
	}
	if got := stringValues(readState.Roles); !reflect.DeepEqual(got, []string{"role-1"}) {
		t.Fatalf("read roles = %#v", got)
	}

	updatePlan := orgGroupPlan(t, schemaResp.Schema, OrgGroupModel{
		Name:        types.StringValue("admins-updated"),
		Description: types.StringNull(),
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
	if got := bodies[0]["members"]; !reflect.DeepEqual(got, []interface{}{"user-1"}) {
		t.Fatalf("create members = %#v", got)
	}
	if _, ok := bodies[0]["roles"]; ok {
		t.Fatalf("create body should not include roles: %#v", bodies[0])
	}
	if got := bodies[1]["roles"]; !reflect.DeepEqual(got, []interface{}{"role-1"}) {
		t.Fatalf("post-create roles = %#v", got)
	}
	if got := bodies[2]["description"]; got != "" {
		t.Fatalf("update description = %#v, want clear string", got)
	}
	if got := bodies[2]["members"]; !reflect.DeepEqual(got, []interface{}{}) {
		t.Fatalf("update members = %#v, want clear list", got)
	}
	if got := bodies[2]["roles"]; !reflect.DeepEqual(got, []interface{}{}) {
		t.Fatalf("update roles = %#v, want clear list", got)
	}
}

func TestOrgGroupReadRemovesMissingGroupAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing group", statusCode: http.StatusNotFound, wantRemove: true},
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
			mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &OrgGroupResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := orgGroupState(t, schemaResp.Schema, OrgGroupModel{
				ID:   types.StringValue("group-123"),
				Name: types.StringValue("admins"),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing group to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestOrgGroupReadPreservesManagedRelationshipsWhenAPIOmitsThem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "group-123",
			"name":        "admins",
			"description": "from api",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgGroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := orgGroupState(t, schemaResp.Schema, OrgGroupModel{
		ID:          types.StringValue("group-123"),
		Name:        types.StringValue("old-name"),
		Description: types.StringValue("old"),
		Members:     []types.String{types.StringValue("user-1")},
		Roles:       []types.String{types.StringValue("role-1")},
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState OrgGroupModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got := stringValues(readState.Members); !reflect.DeepEqual(got, []string{"user-1"}) {
		t.Fatalf("members = %#v", got)
	}
	if got := stringValues(readState.Roles); !reflect.DeepEqual(got, []string{"role-1"}) {
		t.Fatalf("roles = %#v", got)
	}
}

func TestOrgGroupReadIgnoresRemoteRelationshipsWhenStateDoesNotManageThem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "group-123",
			"name":        "admins",
			"description": "from api",
			"members":     []interface{}{"remote-user"},
			"roles":       []interface{}{"remote-role"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgGroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := orgGroupState(t, schemaResp.Schema, OrgGroupModel{
		ID:   types.StringValue("group-123"),
		Name: types.StringValue("old-name"),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState OrgGroupModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if readState.Members != nil {
		t.Fatalf("members = %#v, want nil", readState.Members)
	}
	if readState.Roles != nil {
		t.Fatalf("roles = %#v, want nil", readState.Roles)
	}
}

func TestOrgGroupReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &OrgGroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := orgGroupPlan(t, schemaResp.Schema, OrgGroupModel{
		Name: types.StringValue("admins"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := orgGroupState(t, schemaResp.Schema, OrgGroupModel{
		ID:   types.StringValue("group-123"),
		Name: types.StringValue("admins"),
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

func TestOrgGroupCreateReportsPostCreateUpdateError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "group-123",
			"name": "admins",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("item method = %s, want PUT", r.Method)
		}
		http.Error(w, "post-create update failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgGroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := orgGroupPlan(t, schemaResp.Schema, OrgGroupModel{
		Name:  types.StringValue("admins"),
		Roles: []types.String{types.StringValue("role-1")},
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected post-create update diagnostics")
	}
}

func TestOrgGroupDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgGroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := orgGroupState(t, schemaResp.Schema, OrgGroupModel{
		ID:   types.StringValue("group-123"),
		Name: types.StringValue("admins"),
	})

	resp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", resp.Diagnostics)
	}
}

func TestOrgGroupImportStateSetsID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewOrgGroupResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: orgGroupState(t, schemaResp.Schema, OrgGroupModel{
		ID:   types.StringValue("placeholder"),
		Name: types.StringValue("admins"),
	})}

	(&OrgGroupResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "group-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported OrgGroupModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.ID.ValueString(), "group-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func orgGroupPlan(t *testing.T, schema resourceschema.Schema, model OrgGroupModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func orgGroupState(t *testing.T, schema resourceschema.Schema, model OrgGroupModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
