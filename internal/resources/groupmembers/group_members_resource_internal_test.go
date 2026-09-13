package groupmembers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestGroupMembersMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewGroupMembersResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_group_members" {
		t.Fatalf("type name = %q, want graviteeam_group_members", resp.TypeName)
	}
}

func TestGroupMembersSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewGroupMembersResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "group_id", "members"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if !strings.Contains(resp.Schema.Description, "complete membership list") {
		t.Fatalf("schema description = %q, want complete-list ownership hint", resp.Schema.Description)
	}
}

func TestGroupMembersImportRejectsEmptyComponents(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewGroupMembersResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	for _, id := range []string{"/group-1", "domain-1/", "domain-1/ ", "/"} {
		state := tfsdk.State{Schema: schemaResp.Schema}
		if diags := state.Set(context.Background(), &GroupMembersModel{
			DomainID: types.StringValue("placeholder"),
			GroupID:  types.StringValue("placeholder"),
			Members:  []types.String{types.StringValue("member-1")},
		}); diags.HasError() {
			t.Fatalf("set state: %#v", diags)
		}
		resp := &resource.ImportStateResponse{State: state}
		NewGroupMembersResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{ID: id}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("%q: expected invalid import diagnostics", id)
		}
	}
}

func TestGroupMembersConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GroupMembersResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestGroupMembersConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&GroupMembersResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestGroupMembersConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GroupMembersResource{}
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

	model := GroupMembersModel{
		DomainID: types.StringValue("domain-1"),
		GroupID:  types.StringValue("group-1"),
		Members:  []types.String{types.StringValue("old-user")},
	}

	readIntoModel(&model, []string{"user-1", "user-2"})

	gotMembers := stringValues(model.Members)
	wantMembers := []string{"user-1", "user-2"}
	if !reflect.DeepEqual(gotMembers, wantMembers) {
		t.Fatalf("members = %#v, want %#v", gotMembers, wantMembers)
	}
}

func TestDiffMembers(t *testing.T) {
	t.Parallel()

	toAdd, toRemove := diffMembers(
		[]types.String{types.StringValue("user-c"), types.StringValue("user-b"), types.StringValue("user-c")},
		[]types.String{types.StringValue("user-a"), types.StringValue("user-b"), types.StringValue("user-a")},
	)

	if want := []string{"user-c"}; !reflect.DeepEqual(toAdd, want) {
		t.Fatalf("toAdd = %#v, want %#v", toAdd, want)
	}
	if want := []string{"user-a"}; !reflect.DeepEqual(toRemove, want) {
		t.Fatalf("toRemove = %#v, want %#v", toRemove, want)
	}
}

func TestGroupMembersCRUDReconcilesOnlyMembershipDiff(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/members", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("page") != "0" || r.URL.Query().Get("size") != "100" {
			t.Fatalf("unexpected group members collection request: %s %s", r.Method, r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{{"id": "user-b"}, {"id": "user-c"}},
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/members/", func(w http.ResponseWriter, r *http.Request) {
		memberID := strings.TrimPrefix(r.URL.Path, "/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/members/")
		switch r.Method {
		case http.MethodPost:
			operations = append(operations, "add:"+memberID)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			operations = append(operations, "remove:"+memberID)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected member method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GroupMembersResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := groupMembersPlan(t, schemaResp.Schema, GroupMembersModel{
		DomainID: types.StringValue("domain-123"),
		GroupID:  types.StringValue("group-123"),
		Members:  []types.String{types.StringValue("user-a"), types.StringValue("user-b")},
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
	var readState GroupMembersModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := stringValues(readState.Members), []string{"user-b", "user-c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("read members = %#v, want %#v", got, want)
	}

	updatePlan := groupMembersPlan(t, schemaResp.Schema, GroupMembersModel{
		DomainID: types.StringValue("domain-123"),
		GroupID:  types.StringValue("group-123"),
		Members:  []types.String{types.StringValue("user-c"), types.StringValue("user-d")},
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
	wantOperations := []string{
		"add:user-a",
		"add:user-b",
		"remove:user-b",
		"add:user-d",
		"remove:user-c",
		"remove:user-d",
	}
	if !reflect.DeepEqual(operations, wantOperations) {
		t.Fatalf("operations = %#v, want %#v", operations, wantOperations)
	}
}

func TestGroupMembersReadRemovesMissingGroupAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantRemove bool
	}{
		{
			name:       "missing group",
			statusCode: http.StatusNotFound,
			wantRemove: true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantRemove: false,
		},
		{
			name:       "server error mentioning 404",
			statusCode: http.StatusInternalServerError,
			body:       "upstream returned 404",
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/members", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed: "+tt.body, tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &GroupMembersResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := groupMembersState(t, schemaResp.Schema, GroupMembersModel{
				DomainID: types.StringValue("domain-123"),
				GroupID:  types.StringValue("group-123"),
				Members:  []types.String{types.StringValue("user-a")},
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

func TestGroupMembersLifecycleStopsOnInvalidPlanOrState(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewGroupMembersResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	invalidPlan := groupMembersInvalidPlan(schemaResp.Schema)
	invalidState := groupMembersInvalidState(schemaResp.Schema)
	validPlan := groupMembersPlan(t, schemaResp.Schema, GroupMembersModel{
		DomainID: types.StringValue("domain-123"),
		GroupID:  types.StringValue("group-123"),
		Members:  []types.String{types.StringValue("user-a")},
	})

	t.Run("create invalid plan", func(t *testing.T) {
		resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&GroupMembersResource{}).Create(context.Background(), resource.CreateRequest{Plan: invalidPlan}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected create diagnostics")
		}
	})
	t.Run("read invalid state", func(t *testing.T) {
		resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&GroupMembersResource{}).Read(context.Background(), resource.ReadRequest{State: invalidState}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected read diagnostics")
		}
	})
	t.Run("update invalid plan", func(t *testing.T) {
		resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&GroupMembersResource{}).Update(context.Background(), resource.UpdateRequest{
			Plan:  invalidPlan,
			State: groupMembersState(t, schemaResp.Schema, GroupMembersModel{}),
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected update diagnostics")
		}
	})
	t.Run("update invalid state", func(t *testing.T) {
		resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		(&GroupMembersResource{}).Update(context.Background(), resource.UpdateRequest{
			Plan:  validPlan,
			State: invalidState,
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected update diagnostics")
		}
	})
	t.Run("delete invalid state", func(t *testing.T) {
		resp := &resource.DeleteResponse{}
		(&GroupMembersResource{}).Delete(context.Background(), resource.DeleteRequest{State: invalidState}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected delete diagnostics")
		}
	})
}

func TestGroupMembersReportsCreateUpdateAndDeleteErrors(t *testing.T) {
	tests := []struct {
		name          string
		failingMethod string
		failingMember string
		run           func(*GroupMembersResource, resourceschema.Schema)
	}{
		{
			name:          "create add error",
			failingMethod: http.MethodPost,
			failingMember: "user-a",
			run: func(resourceUnderTest *GroupMembersResource, schema resourceschema.Schema) {
				plan := groupMembersPlan(t, schema, GroupMembersModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Members:  []types.String{types.StringValue("user-a")},
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
			failingMember: "user-a",
			run: func(resourceUnderTest *GroupMembersResource, schema resourceschema.Schema) {
				plan := groupMembersPlan(t, schema, GroupMembersModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Members:  []types.String{types.StringValue("user-b")},
				})
				state := groupMembersState(t, schema, GroupMembersModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Members:  []types.String{types.StringValue("user-a")},
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
			failingMember: "user-b",
			run: func(resourceUnderTest *GroupMembersResource, schema resourceschema.Schema) {
				plan := groupMembersPlan(t, schema, GroupMembersModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Members:  []types.String{types.StringValue("user-b")},
				})
				state := groupMembersState(t, schema, GroupMembersModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Members:  nil,
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
			failingMember: "user-a",
			run: func(resourceUnderTest *GroupMembersResource, schema resourceschema.Schema) {
				state := groupMembersState(t, schema, GroupMembersModel{
					DomainID: types.StringValue("domain-123"),
					GroupID:  types.StringValue("group-123"),
					Members:  []types.String{types.StringValue("user-a")},
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/members/", func(w http.ResponseWriter, r *http.Request) {
				memberID := strings.TrimPrefix(r.URL.Path, "/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/groups/group-123/members/")
				if r.Method == tt.failingMethod && memberID == tt.failingMember {
					http.Error(w, "membership operation failed", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &GroupMembersResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			tt.run(resourceUnderTest, schemaResp.Schema)
		})
	}
}

func TestGroupMembersImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&GroupMembersResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain/group/extra",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestGroupMembersImportStateSetsDomainAndGroup(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewGroupMembersResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: groupMembersState(t, schemaResp.Schema, GroupMembersModel{
		DomainID: types.StringValue("placeholder"),
		GroupID:  types.StringValue("placeholder"),
		Members:  []types.String{types.StringValue("user-a")},
	})}

	(&GroupMembersResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/group-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported GroupMembersModel
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

func groupMembersPlan(t *testing.T, schema resourceschema.Schema, model GroupMembersModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func groupMembersInvalidPlan(schema resourceschema.Schema) tfsdk.Plan {
	return tfsdk.Plan{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.Number,
				"group_id":  tftypes.String,
				"members":   tftypes.List{ElementType: tftypes.String},
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.Number, 123),
				"group_id":  tftypes.NewValue(tftypes.String, "group-123"),
				"members":   tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "user-a")}),
			},
		),
		Schema: schema,
	}
}

func groupMembersState(t *testing.T, schema resourceschema.Schema, model GroupMembersModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func groupMembersInvalidState(schema resourceschema.Schema) tfsdk.State {
	return tfsdk.State{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id": tftypes.Number,
				"group_id":  tftypes.String,
				"members":   tftypes.List{ElementType: tftypes.String},
			}},
			map[string]tftypes.Value{
				"domain_id": tftypes.NewValue(tftypes.Number, 123),
				"group_id":  tftypes.NewValue(tftypes.String, "group-123"),
				"members":   tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "user-a")}),
			},
		),
		Schema: schema,
	}
}
