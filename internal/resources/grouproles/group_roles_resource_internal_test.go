package grouproles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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
	NewGroupRolesResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_group_roles"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewGroupRolesResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "domain_id")
	assertStringAttribute(t, resp.Schema.Attributes, "group_id")
	assertSetAttribute(t, resp.Schema.Attributes, "roles", types.StringType)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&GroupRolesResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&GroupRolesResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		id           string
		wantDomainID string
		wantGroupID  string
		wantOK       bool
	}{
		{
			name:         "valid",
			id:           "domain-1/group-1",
			wantDomainID: "domain-1",
			wantGroupID:  "group-1",
			wantOK:       true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
		{
			name:   "too many segments",
			id:     "domain-1/group-1/extra",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotGroupID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotGroupID != tt.wantGroupID {
				t.Fatalf("group ID = %q, want %q", gotGroupID, tt.wantGroupID)
			}
		})
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	model := GroupRolesModel{
		DomainID: types.StringValue("domain-1"),
		GroupID:  types.StringValue("group-1"),
		Roles:    []types.String{types.StringValue("old-role")},
	}

	readIntoModel(&model, []interface{}{
		map[string]interface{}{"id": "role-1"},
		map[string]interface{}{"name": "missing-id"},
		map[string]interface{}{"id": 123},
		"not-a-map",
		map[string]interface{}{"id": "role-2"},
	})

	gotRoles := stringValues(model.Roles)
	wantRoles := []string{"role-1", "role-2"}
	if !reflect.DeepEqual(gotRoles, wantRoles) {
		t.Fatalf("roles = %#v, want %#v", gotRoles, wantRoles)
	}
}

func TestDiffRoles(t *testing.T) {
	t.Parallel()

	toAdd, toRemove := diffRoles(
		[]types.String{types.StringValue("role-c"), types.StringValue("role-b"), types.StringValue("role-c")},
		[]types.String{types.StringValue("role-a"), types.StringValue("role-b"), types.StringValue("role-a")},
	)

	if want := []string{"role-c"}; !reflect.DeepEqual(toAdd, want) {
		t.Fatalf("toAdd = %#v, want %#v", toAdd, want)
	}
	if want := []string{"role-a"}; !reflect.DeepEqual(toRemove, want) {
		t.Fatalf("toRemove = %#v, want %#v", toRemove, want)
	}
}

func TestGroupRolesCRUDReconcilesOnlyRoleDiff(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	var operations []string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/roles", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "role-b"}, {"id": "role-c"}})
		case http.MethodPost:
			var roleIDs []string
			if err := json.NewDecoder(r.Body).Decode(&roleIDs); err != nil {
				t.Fatalf("decode role assignment body: %v", err)
			}
			operations = append(operations, "set:"+strings.Join(roleIDs, ","))
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		default:
			t.Fatalf("unexpected group roles method %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/roles/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected group role method %s", r.Method)
		}
		roleID := strings.TrimPrefix(r.URL.Path, "/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/roles/")
		operations = append(operations, "remove:"+roleID)
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GroupRolesResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := groupRolesPlan(t, schemaResp.Schema, GroupRolesModel{
		DomainID: types.StringValue("domain-123"),
		GroupID:  types.StringValue("group-123"),
		Roles:    []types.String{types.StringValue("role-a"), types.StringValue("role-b")},
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
	var readState GroupRolesModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := stringValues(readState.Roles), []string{"role-b", "role-c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("read roles = %#v, want %#v", got, want)
	}

	updatePlan := groupRolesPlan(t, schemaResp.Schema, GroupRolesModel{
		DomainID: types.StringValue("domain-123"),
		GroupID:  types.StringValue("group-123"),
		Roles:    []types.String{types.StringValue("role-c"), types.StringValue("role-d")},
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

	wantOperations := []string{
		"set:role-a,role-b",
		"remove:role-b",
		"set:role-d",
		"remove:role-c",
		"remove:role-d",
	}
	if !reflect.DeepEqual(operations, wantOperations) {
		t.Fatalf("operations = %#v, want %#v", operations, wantOperations)
	}
}

func TestGroupRolesReadRemovesMissingGroupAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{"missing group", http.StatusNotFound, true},
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/roles", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &GroupRolesResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := groupRolesState(t, schemaResp.Schema, GroupRolesModel{
				DomainID: types.StringValue("domain-123"),
				GroupID:  types.StringValue("group-123"),
				Roles:    []types.String{types.StringValue("role-a")},
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing group roles to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestGroupRolesLifecycleStopsOnInvalidPlanOrState(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewGroupRolesResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	invalidPlan := groupRolesInvalidPlan(schemaResp.Schema)
	invalidState := groupRolesInvalidState(schemaResp.Schema)
	validPlan := groupRolesPlan(t, schemaResp.Schema, GroupRolesModel{
		DomainID: types.StringValue("domain-123"),
		GroupID:  types.StringValue("group-123"),
		Roles:    []types.String{types.StringValue("role-a")},
	})

	t.Run("create invalid plan", func(t *testing.T) {
		resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&GroupRolesResource{}).Create(context.Background(), resource.CreateRequest{Plan: invalidPlan}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected create diagnostics")
		}
	})
	t.Run("read invalid state", func(t *testing.T) {
		resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&GroupRolesResource{}).Read(context.Background(), resource.ReadRequest{State: invalidState}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected read diagnostics")
		}
	})
	t.Run("update invalid plan", func(t *testing.T) {
		resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&GroupRolesResource{}).Update(context.Background(), resource.UpdateRequest{
			Plan:  invalidPlan,
			State: groupRolesState(t, schemaResp.Schema, GroupRolesModel{}),
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected update diagnostics")
		}
	})
	t.Run("update invalid state", func(t *testing.T) {
		resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&GroupRolesResource{}).Update(context.Background(), resource.UpdateRequest{
			Plan:  validPlan,
			State: invalidState,
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected update diagnostics")
		}
	})
	t.Run("delete invalid state", func(t *testing.T) {
		resp := &resource.DeleteResponse{}
		(&GroupRolesResource{}).Delete(context.Background(), resource.DeleteRequest{State: invalidState}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected delete diagnostics")
		}
	})
}

func TestGroupRolesReportsCreateUpdateAndDeleteErrors(t *testing.T) {
	tests := []struct {
		name          string
		failingMethod string
		failingRole   string
		run           func(*GroupRolesResource, resourceschema.Schema)
	}{
		{
			name:          "create set error",
			failingMethod: http.MethodPost,
			run: func(resourceUnderTest *GroupRolesResource, schema resourceschema.Schema) {
				plan := groupRolesPlan(t, schema, GroupRolesModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Roles:    []types.String{types.StringValue("role-a")},
				})
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected create diagnostics")
				}
			},
		},
		{
			name:          "update remove error",
			failingMethod: http.MethodDelete,
			failingRole:   "role-a",
			run: func(resourceUnderTest *GroupRolesResource, schema resourceschema.Schema) {
				plan := groupRolesPlan(t, schema, GroupRolesModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Roles:    []types.String{types.StringValue("role-b")},
				})
				state := groupRolesState(t, schema, GroupRolesModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Roles:    []types.String{types.StringValue("role-a")},
				})
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected update diagnostics")
				}
			},
		},
		{
			name:          "update add error",
			failingMethod: http.MethodPost,
			run: func(resourceUnderTest *GroupRolesResource, schema resourceschema.Schema) {
				plan := groupRolesPlan(t, schema, GroupRolesModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Roles:    []types.String{types.StringValue("role-b")},
				})
				state := groupRolesState(t, schema, GroupRolesModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Roles:    nil,
				})
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected update diagnostics")
				}
			},
		},
		{
			name:          "delete remove error",
			failingMethod: http.MethodDelete,
			failingRole:   "role-a",
			run: func(resourceUnderTest *GroupRolesResource, schema resourceschema.Schema) {
				state := groupRolesState(t, schema, GroupRolesModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Roles:    []types.String{types.StringValue("role-a")},
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/roles", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == tt.failingMethod && tt.failingRole == "" {
					http.Error(w, "set failed", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/roles/", func(w http.ResponseWriter, r *http.Request) {
				roleID := strings.TrimPrefix(r.URL.Path, "/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/roles/")
				if r.Method == tt.failingMethod && roleID == tt.failingRole {
					http.Error(w, "remove failed", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &GroupRolesResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			tt.run(resourceUnderTest, schemaResp.Schema)
		})
	}
}

func TestGroupRolesImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&GroupRolesResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain/group/extra",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestGroupRolesConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GroupRolesResource{}
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

func TestGroupRolesImportStateSetsDomainAndGroup(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewGroupRolesResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: groupRolesState(t, schemaResp.Schema, GroupRolesModel{
		DomainID: types.StringValue("placeholder"),
		GroupID:  types.StringValue("placeholder"),
		Roles:    []types.String{types.StringValue("role-a")},
	})}

	(&GroupRolesResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/group-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported GroupRolesModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.GroupID.ValueString(), "group-123"; got != want {
		t.Fatalf("group id = %q, want %q", got, want)
	}
}

func groupRolesPlan(t *testing.T, schema resourceschema.Schema, model GroupRolesModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func groupRolesInvalidPlan(schema resourceschema.Schema) tfsdk.Plan {
	return tfsdk.Plan{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.Number,
				"group_id":  tftypes.String,
				"roles":     tftypes.Set{ElementType: tftypes.String},
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.Number, 123),
				"group_id":  tftypes.NewValue(tftypes.String, "group-123"),
				"roles":     tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "role-a")}),
			},
		),
		Schema: schema,
	}
}

func groupRolesState(t *testing.T, schema resourceschema.Schema, model GroupRolesModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func groupRolesInvalidState(schema resourceschema.Schema) tfsdk.State {
	return tfsdk.State{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.Number,
				"group_id":  tftypes.String,
				"roles":     tftypes.Set{ElementType: tftypes.String},
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.Number, 123),
				"group_id":  tftypes.NewValue(tftypes.String, "group-123"),
				"roles":     tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "role-a")}),
			},
		),
		Schema: schema,
	}
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if !attr.Required || attr.Optional || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
}

func assertSetAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, elemType attr.Type) {
	t.Helper()

	attr, ok := attrs[name].(schema.SetAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.SetAttribute", name, attrs[name])
	}
	if !attr.Required || attr.Optional || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
	if !reflect.DeepEqual(attr.ElementType, elemType) {
		t.Fatalf("%s element type = %#v, want %#v", name, attr.ElementType, elemType)
	}
}
