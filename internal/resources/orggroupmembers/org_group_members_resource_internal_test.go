package orggroupmembers

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

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestOrgGroupMembersMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewOrgGroupMembersResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_org_group_members" {
		t.Fatalf("type name = %q, want graviteeam_org_group_members", resp.TypeName)
	}
}

func TestOrgGroupMembersSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgGroupMembersResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"group_id", "members"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if _, ok := resp.Schema.Attributes["domain_id"]; ok {
		t.Fatal("organization group members should not expose domain_id")
	}
	if !strings.Contains(resp.Schema.Description, "complete set") {
		t.Fatalf("schema description = %q, want complete-set ownership hint", resp.Schema.Description)
	}
}

func TestOrgGroupMembersConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgGroupMembersResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestOrgGroupMembersConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgGroupMembersResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestValidImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		id     string
		wantOK bool
	}{
		{
			name:   "valid",
			id:     "group-1",
			wantOK: true,
		},
		{
			name:   "empty",
			id:     "",
			wantOK: false,
		},
		{
			name:   "blank",
			id:     "   ",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := validImportID(tt.id); got != tt.wantOK {
				t.Fatalf("validImportID(%q) = %t, want %t", tt.id, got, tt.wantOK)
			}
		})
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	model := OrgGroupMembersModel{
		GroupID: types.StringValue("group-1"),
		Members: []types.String{types.StringValue("old-user")},
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

func TestOrgGroupMembersCRUDReconcilesOnlyMembershipDiff(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123/members", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("page") != "0" || r.URL.Query().Get("size") != "100" {
			t.Fatalf("unexpected org group members collection request: %s %s", r.Method, r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{{"id": "user-b"}, {"id": "user-c"}},
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123/members/", func(w http.ResponseWriter, r *http.Request) {
		memberID := strings.TrimPrefix(r.URL.Path, "/management/organizations/DEFAULT/groups/group-123/members/")
		switch r.Method {
		case http.MethodPost:
			operations = append(operations, "add:"+memberID)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			operations = append(operations, "remove:"+memberID)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected org member method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgGroupMembersResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgGroupMembersPlan(t, schemaResp.Schema, OrgGroupMembersModel{
		GroupID: types.StringValue("group-123"),
		Members: []types.String{types.StringValue("user-a"), types.StringValue("user-b")},
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
	var readState OrgGroupMembersModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := stringValues(readState.Members), []string{"user-b", "user-c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("read members = %#v, want %#v", got, want)
	}

	updatePlan := orgGroupMembersPlan(t, schemaResp.Schema, OrgGroupMembersModel{
		GroupID: types.StringValue("group-123"),
		Members: []types.String{types.StringValue("user-c"), types.StringValue("user-d")},
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

func TestOrgGroupMembersReadRemovesMissingGroupAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123/members", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &OrgGroupMembersResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := orgGroupMembersState(t, schemaResp.Schema, OrgGroupMembersModel{
				GroupID: types.StringValue("group-123"),
				Members: []types.String{types.StringValue("user-a")},
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

func TestOrgGroupMembersReportsCreateUpdateAndDeleteErrors(t *testing.T) {
	tests := []struct {
		name          string
		failingMethod string
		failingMember string
		run           func(*OrgGroupMembersResource, resourceschema.Schema)
	}{
		{
			name:          "create add error",
			failingMethod: http.MethodPost,
			failingMember: "user-a",
			run: func(resourceUnderTest *OrgGroupMembersResource, schema resourceschema.Schema) {
				plan := orgGroupMembersPlan(t, schema, OrgGroupMembersModel{
					GroupID: types.StringValue("group-123"),
					Members: []types.String{types.StringValue("user-a")},
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
			run: func(resourceUnderTest *OrgGroupMembersResource, schema resourceschema.Schema) {
				plan := orgGroupMembersPlan(t, schema, OrgGroupMembersModel{
					GroupID: types.StringValue("group-123"),
					Members: []types.String{types.StringValue("user-b")},
				})
				state := orgGroupMembersState(t, schema, OrgGroupMembersModel{
					GroupID: types.StringValue("group-123"),
					Members: []types.String{types.StringValue("user-a")},
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
			run: func(resourceUnderTest *OrgGroupMembersResource, schema resourceschema.Schema) {
				plan := orgGroupMembersPlan(t, schema, OrgGroupMembersModel{
					GroupID: types.StringValue("group-123"),
					Members: []types.String{types.StringValue("user-b")},
				})
				state := orgGroupMembersState(t, schema, OrgGroupMembersModel{
					GroupID: types.StringValue("group-123"),
					Members: nil,
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
			run: func(resourceUnderTest *OrgGroupMembersResource, schema resourceschema.Schema) {
				state := orgGroupMembersState(t, schema, OrgGroupMembersModel{
					GroupID: types.StringValue("group-123"),
					Members: []types.String{types.StringValue("user-a")},
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
			mux.HandleFunc("/management/organizations/DEFAULT/groups/group-123/members/", func(w http.ResponseWriter, r *http.Request) {
				memberID := strings.TrimPrefix(r.URL.Path, "/management/organizations/DEFAULT/groups/group-123/members/")
				if r.Method == tt.failingMethod && memberID == tt.failingMember {
					http.Error(w, "membership operation failed", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &OrgGroupMembersResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			tt.run(resourceUnderTest, schemaResp.Schema)
		})
	}
}

func TestOrgGroupMembersImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&OrgGroupMembersResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "   ",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestOrgGroupMembersImportStateSetsGroupID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewOrgGroupMembersResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: orgGroupMembersState(t, schemaResp.Schema, OrgGroupMembersModel{
		GroupID: types.StringValue("placeholder"),
		Members: []types.String{
			types.StringValue("user-a"),
		},
	})}

	(&OrgGroupMembersResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "group-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported OrgGroupMembersModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.GroupID.ValueString(), "group-123"; got != want {
		t.Fatalf("group id = %q, want %q", got, want)
	}
}

func orgGroupMembersPlan(t *testing.T, schema resourceschema.Schema, model OrgGroupMembersModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func orgGroupMembersState(t *testing.T, schema resourceschema.Schema, model OrgGroupMembersModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
