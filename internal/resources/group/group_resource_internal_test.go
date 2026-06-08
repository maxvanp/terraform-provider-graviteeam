package group

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
	NewGroupResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_group"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewGroupResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "description", false, true, false)
	assertListAttribute(t, resp.Schema.Attributes, "members", types.StringType)
	assertListAttribute(t, resp.Schema.Attributes, "roles", types.StringType)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&GroupResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&GroupResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestBuildUpdateBodyClearsRemovedGroupLists(t *testing.T) {
	plan := GroupModel{
		Name: types.StringValue("updated"),
	}
	state := GroupModel{
		Name:        types.StringValue("updated"),
		Description: types.StringValue("old description"),
		Members: []types.String{
			types.StringValue("user-id"),
		},
		Roles: []types.String{
			types.StringValue("role-id"),
		},
	}

	body := (&GroupResource{}).buildUpdateBody(plan, &state)

	if got := body["description"]; got != nil {
		t.Fatalf("description = %#v, want omitted when removed", got)
	}
	members, ok := body["members"].([]string)
	if !ok {
		t.Fatalf("members = %#v, want []string", body["members"])
	}
	if len(members) != 0 {
		t.Fatalf("members length = %d, want 0", len(members))
	}
	roles, ok := body["roles"].([]string)
	if !ok {
		t.Fatalf("roles = %#v, want []string", body["roles"])
	}
	if len(roles) != 0 {
		t.Fatalf("roles length = %d, want 0", len(roles))
	}
}

func TestReadIntoModelMapsGroupResponse(t *testing.T) {
	model := GroupModel{}

	(&GroupResource{}).readIntoModel(&model, map[string]interface{}{
		"id":          "group-id",
		"name":        "group-name",
		"description": "description",
		"members":     []interface{}{"user-1", "user-2"},
		"roles":       []interface{}{"role-1"},
	})

	if model.ID.ValueString() != "group-id" {
		t.Fatalf("id = %q, want group-id", model.ID.ValueString())
	}
	if model.Name.ValueString() != "group-name" {
		t.Fatalf("name = %q, want group-name", model.Name.ValueString())
	}
	if model.Description.ValueString() != "description" {
		t.Fatalf("description = %q, want description", model.Description.ValueString())
	}
	if len(model.Members) != 2 || model.Members[0].ValueString() != "user-1" || model.Members[1].ValueString() != "user-2" {
		t.Fatalf("members = %#v, want user-1,user-2", model.Members)
	}
	if len(model.Roles) != 1 || model.Roles[0].ValueString() != "role-1" {
		t.Fatalf("roles = %#v, want role-1", model.Roles)
	}
}

func TestGroupCRUDPreservesAndClearsOptionalRelationships(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected group collection method %s", r.Method)
		}
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "group-123",
				"name":        "group-from-api",
				"description": "from api",
			})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "group-123",
				"name":        body["name"],
				"description": body["description"],
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected group item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := groupPlan(t, schemaResp.Schema, GroupModel{
		DomainID:    types.StringValue("domain-123"),
		Name:        types.StringValue("group-name"),
		Description: types.StringValue("created"),
		Members:     []types.String{types.StringValue("user-a")},
		Roles:       []types.String{types.StringValue("role-a")},
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState GroupModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if got, want := stringValues(createState.Members), []string{"user-a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("created members = %#v, want %#v; request bodies = %#v", got, want, bodies)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState GroupModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := stringValues(readState.Members), []string{"user-a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("preserved members = %#v, want %#v", got, want)
	}
	if got, want := stringValues(readState.Roles), []string{"role-a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("preserved roles = %#v, want %#v", got, want)
	}

	updatePlan := groupPlan(t, schemaResp.Schema, GroupModel{
		DomainID:    types.StringValue("domain-123"),
		Name:        types.StringValue("group-updated"),
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

	if len(bodies) != 3 {
		t.Fatalf("request bodies = %#v, want create plus two updates", bodies)
	}
	if got, want := bodies[0]["members"], []interface{}{"user-a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("create members = %#v, want %#v", got, want)
	}
	if got, want := bodies[1]["roles"], []interface{}{"role-a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("create follow-up roles = %#v, want %#v", got, want)
	}
	if got, want := bodies[2]["members"], []interface{}{}; !reflect.DeepEqual(got, want) {
		t.Fatalf("update members = %#v, want clear list %#v", got, want)
	}
	if got, want := bodies[2]["roles"], []interface{}{}; !reflect.DeepEqual(got, want) {
		t.Fatalf("update roles = %#v, want clear list %#v", got, want)
	}
	if _, ok := bodies[2]["description"]; ok {
		t.Fatalf("update body should omit removed description: %#v", bodies[2])
	}
}

func TestGroupReadRemovesMissingGroupAndReportsErrors(t *testing.T) {
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &GroupResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := groupState(t, schemaResp.Schema, GroupModel{
				ID:       types.StringValue("group-123"),
				DomainID: types.StringValue("domain-123"),
				Name:     types.StringValue("group-name"),
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

func TestGroupReadPreservesManagedRelationshipsWhenAPIOmitsThem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "group-123",
			"name":        "group-name",
			"description": "from api",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := groupState(t, schemaResp.Schema, GroupModel{
		ID:          types.StringValue("group-123"),
		DomainID:    types.StringValue("domain-123"),
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
	var readState GroupModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if len(readState.Members) != 1 || readState.Members[0].ValueString() != "user-1" {
		t.Fatalf("members = %#v", readState.Members)
	}
	if len(readState.Roles) != 1 || readState.Roles[0].ValueString() != "role-1" {
		t.Fatalf("roles = %#v", readState.Roles)
	}
}

func TestGroupReadIgnoresRemoteRelationshipsWhenStateDoesNotManageThem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "group-123",
			"name":        "group-name",
			"description": "from api",
			"members":     []interface{}{"remote-user"},
			"roles":       []interface{}{"remote-role"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := groupState(t, schemaResp.Schema, GroupModel{
		ID:       types.StringValue("group-123"),
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("old-name"),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState GroupModel
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

func TestGroupUpdatePreservesPlannedRelationshipsWhenAPIOmitsThem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "group-123",
			"name":        "group-updated",
			"description": "updated",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := groupPlan(t, schemaResp.Schema, GroupModel{
		DomainID:    types.StringValue("domain-123"),
		Name:        types.StringValue("group-updated"),
		Description: types.StringValue("updated"),
		Members:     []types.String{types.StringValue("user-1")},
		Roles:       []types.String{types.StringValue("role-1")},
	})
	state := groupState(t, schemaResp.Schema, GroupModel{
		ID:       types.StringValue("group-123"),
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("group-old"),
	})

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	var updated GroupModel
	if diags := updateResp.State.Get(context.Background(), &updated); diags.HasError() {
		t.Fatalf("get update state: %#v", diags)
	}
	if got, want := stringValues(updated.Members), []string{"user-1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("members = %#v, want %#v", got, want)
	}
	if got, want := stringValues(updated.Roles), []string{"role-1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("roles = %#v, want %#v", got, want)
	}
}

func TestGroupReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &GroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := groupPlan(t, schemaResp.Schema, GroupModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("group-name"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := groupState(t, schemaResp.Schema, GroupModel{
		ID:       types.StringValue("group-123"),
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("group-name"),
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

func TestGroupCreateReadUpdateAndDeleteReportInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GroupResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"domain_id":   tftypes.Number,
			"name":        tftypes.String,
			"description": tftypes.String,
			"members":     tftypes.List{ElementType: tftypes.String},
			"roles":       tftypes.List{ElementType: tftypes.String},
		}},
		map[string]tftypes.Value{
			"id":          tftypes.NewValue(tftypes.String, "group-123"),
			"domain_id":   tftypes.NewValue(tftypes.Number, 123),
			"name":        tftypes.NewValue(tftypes.String, "group-name"),
			"description": tftypes.NewValue(tftypes.String, nil),
			"members":     tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"roles":       tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		},
	)

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
	}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, readResp)
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

	validPlan := groupPlan(t, schemaResp.Schema, GroupModel{
		DomainID:    types.StringValue("domain-123"),
		Name:        types.StringValue("group-name"),
		Description: types.StringValue("created"),
		Members:     []types.String{types.StringValue("user-123")},
		Roles:       []types.String{types.StringValue("role-123")},
	})
	updateResp = &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  validPlan,
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update state diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestGroupCreateReportsPostCreateUpdateError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "group-123",
			"name": "group-name",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("item method = %s, want PUT", r.Method)
		}
		http.Error(w, "post-create update failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GroupResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := groupPlan(t, schemaResp.Schema, GroupModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("group-name"),
		Roles:    []types.String{types.StringValue("role-a")},
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected post-create update diagnostics")
	}
}

func TestGroupImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&GroupResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestGroupConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GroupResource{}
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

func TestGroupImportStateSetsDomainAndID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewGroupResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: groupState(t, schemaResp.Schema, GroupModel{
		ID:       types.StringValue("placeholder"),
		DomainID: types.StringValue("placeholder"),
		Name:     types.StringValue("group-name"),
	})}

	(&GroupResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/group-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported GroupModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.ID.ValueString(), "group-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func groupPlan(t *testing.T, schema resourceschema.Schema, model GroupModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func groupState(t *testing.T, schema resourceschema.Schema, model GroupModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func stringValues(values []types.String) []string {
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
