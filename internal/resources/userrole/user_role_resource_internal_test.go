package userrole

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
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewUserRoleResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_user_role"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewUserRoleResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "domain_id")
	assertStringAttribute(t, resp.Schema.Attributes, "user_id")
	assertSetAttribute(t, resp.Schema.Attributes, "roles", types.StringType)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&UserRoleResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&UserRoleResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

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
		wantUserID   string
		wantOK       bool
	}{
		{
			name:         "valid",
			id:           "domain-1/user-1",
			wantDomainID: "domain-1",
			wantUserID:   "user-1",
			wantOK:       true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
		{
			name:   "too many segments",
			id:     "domain-1/user-1/extra",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotUserID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotUserID != tt.wantUserID {
				t.Fatalf("user ID = %q, want %q", gotUserID, tt.wantUserID)
			}
		})
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	model := UserRoleModel{
		DomainID: types.StringValue("domain-1"),
		UserID:   types.StringValue("user-1"),
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

func TestUserRoleCRUDReconcilesOnlyRoleDiff(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/roles", func(w http.ResponseWriter, r *http.Request) {
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
			t.Fatalf("unexpected user roles method %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/roles/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected user role method %s", r.Method)
		}
		roleID := strings.TrimPrefix(r.URL.Path, "/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/roles/")
		operations = append(operations, "remove:"+roleID)
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserRoleResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := userRolePlan(t, schemaResp.Schema, UserRoleModel{
		DomainID: types.StringValue("domain-123"),
		UserID:   types.StringValue("user-123"),
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
	var readState UserRoleModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := stringValues(readState.Roles), []string{"role-b", "role-c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("read roles = %#v, want %#v", got, want)
	}

	updatePlan := userRolePlan(t, schemaResp.Schema, UserRoleModel{
		DomainID: types.StringValue("domain-123"),
		UserID:   types.StringValue("user-123"),
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

func TestUserRoleReadRemovesMissingUserAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{"missing user", http.StatusNotFound, true},
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/roles", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &UserRoleResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := userRoleState(t, schemaResp.Schema, UserRoleModel{
				DomainID: types.StringValue("domain-123"),
				UserID:   types.StringValue("user-123"),
				Roles:    []types.String{types.StringValue("role-a")},
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing user role state to remove resource, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestUserRoleLifecycleStopsOnInvalidPlanOrState(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewUserRoleResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	invalidPlan := userRoleInvalidPlan(schemaResp.Schema)
	invalidState := userRoleInvalidState(schemaResp.Schema)
	validPlan := userRolePlan(t, schemaResp.Schema, UserRoleModel{
		DomainID: types.StringValue("domain-123"),
		UserID:   types.StringValue("user-123"),
		Roles:    []types.String{types.StringValue("role-a")},
	})

	t.Run("create invalid plan", func(t *testing.T) {
		resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&UserRoleResource{}).Create(context.Background(), resource.CreateRequest{Plan: invalidPlan}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected create diagnostics")
		}
	})
	t.Run("read invalid state", func(t *testing.T) {
		resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&UserRoleResource{}).Read(context.Background(), resource.ReadRequest{State: invalidState}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected read diagnostics")
		}
	})
	t.Run("update invalid plan", func(t *testing.T) {
		resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&UserRoleResource{}).Update(context.Background(), resource.UpdateRequest{
			Plan:  invalidPlan,
			State: userRoleState(t, schemaResp.Schema, UserRoleModel{}),
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected update diagnostics")
		}
	})
	t.Run("update invalid state", func(t *testing.T) {
		resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&UserRoleResource{}).Update(context.Background(), resource.UpdateRequest{
			Plan:  validPlan,
			State: invalidState,
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected update diagnostics")
		}
	})
	t.Run("delete invalid state", func(t *testing.T) {
		resp := &resource.DeleteResponse{}
		(&UserRoleResource{}).Delete(context.Background(), resource.DeleteRequest{State: invalidState}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected delete diagnostics")
		}
	})
}

func TestUserRoleReportsCreateUpdateAndDeleteErrors(t *testing.T) {
	tests := []struct {
		name          string
		failingMethod string
		failingRole   string
		run           func(*UserRoleResource, resourceschema.Schema)
	}{
		{
			name:          "create set error",
			failingMethod: http.MethodPost,
			run: func(resourceUnderTest *UserRoleResource, schema resourceschema.Schema) {
				plan := userRolePlan(t, schema, UserRoleModel{
					DomainID: types.StringValue("domain-123"),
					UserID:   types.StringValue("user-123"),
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
			run: func(resourceUnderTest *UserRoleResource, schema resourceschema.Schema) {
				plan := userRolePlan(t, schema, UserRoleModel{
					DomainID: types.StringValue("domain-123"),
					UserID:   types.StringValue("user-123"),
					Roles:    []types.String{types.StringValue("role-b")},
				})
				state := userRoleState(t, schema, UserRoleModel{
					DomainID: types.StringValue("domain-123"),
					UserID:   types.StringValue("user-123"),
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
			run: func(resourceUnderTest *UserRoleResource, schema resourceschema.Schema) {
				plan := userRolePlan(t, schema, UserRoleModel{
					DomainID: types.StringValue("domain-123"),
					UserID:   types.StringValue("user-123"),
					Roles:    []types.String{types.StringValue("role-b")},
				})
				state := userRoleState(t, schema, UserRoleModel{
					DomainID: types.StringValue("domain-123"),
					UserID:   types.StringValue("user-123"),
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
			run: func(resourceUnderTest *UserRoleResource, schema resourceschema.Schema) {
				state := userRoleState(t, schema, UserRoleModel{
					DomainID: types.StringValue("domain-123"),
					UserID:   types.StringValue("user-123"),
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/roles", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == tt.failingMethod && tt.failingRole == "" {
					http.Error(w, "set failed", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/roles/", func(w http.ResponseWriter, r *http.Request) {
				roleID := strings.TrimPrefix(r.URL.Path, "/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/roles/")
				if r.Method == tt.failingMethod && roleID == tt.failingRole {
					http.Error(w, "remove failed", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &UserRoleResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			tt.run(resourceUnderTest, schemaResp.Schema)
		})
	}
}

func TestUserRoleImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&UserRoleResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain/user/extra",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestUserRoleConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &UserRoleResource{}
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

func TestUserRoleImportStateSetsDomainAndUser(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewUserRoleResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: userRoleState(t, schemaResp.Schema, UserRoleModel{
		DomainID: types.StringValue("placeholder"),
		UserID:   types.StringValue("placeholder"),
		Roles:    []types.String{types.StringValue("role-a")},
	})}

	(&UserRoleResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/user-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported UserRoleModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.UserID.ValueString(), "user-123"; got != want {
		t.Fatalf("user id = %q, want %q", got, want)
	}
}

func userRolePlan(t *testing.T, schema resourceschema.Schema, model UserRoleModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func userRoleInvalidPlan(schema resourceschema.Schema) tfsdk.Plan {
	return tfsdk.Plan{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.Number,
				"user_id":   tftypes.String,
				"roles":     tftypes.Set{ElementType: tftypes.String},
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.Number, 123),
				"user_id":   tftypes.NewValue(tftypes.String, "user-123"),
				"roles":     tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "role-a")}),
			},
		),
		Schema: schema,
	}
}

func userRoleState(t *testing.T, schema resourceschema.Schema, model UserRoleModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func userRoleInvalidState(schema resourceschema.Schema) tfsdk.State {
	return tfsdk.State{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.Number,
				"user_id":   tftypes.String,
				"roles":     tftypes.Set{ElementType: tftypes.String},
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.Number, 123),
				"user_id":   tftypes.NewValue(tftypes.String, "user-123"),
				"roles":     tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "role-a")}),
			},
		),
		Schema: schema,
	}
}

func assertStringAttribute(t *testing.T, attrs map[string]resourceschema.Attribute, name string) {
	t.Helper()

	attr, ok := attrs[name].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want resourceschema.StringAttribute", name, attrs[name])
	}
	if !attr.Required || attr.Optional || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
}

func assertSetAttribute(t *testing.T, attrs map[string]resourceschema.Attribute, name string, elemType attr.Type) {
	t.Helper()

	attr, ok := attrs[name].(resourceschema.SetAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want resourceschema.SetAttribute", name, attrs[name])
	}
	if !attr.Required || attr.Optional || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
	if !reflect.DeepEqual(attr.ElementType, elemType) {
		t.Fatalf("%s element type = %#v, want %#v", name, attr.ElementType, elemType)
	}
}
