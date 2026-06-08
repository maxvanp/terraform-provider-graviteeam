package protectedresourcemember

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
	NewProtectedResourceMemberResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_protected_resource_member"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewProtectedResourceMemberResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "protected_resource_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "member_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "member_type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "role_id", true, false, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ProtectedResourceMemberResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ProtectedResourceMemberResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ProtectedResourceMemberResource{}
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
		name                    string
		id                      string
		wantDomainID            string
		wantProtectedResourceID string
		wantMemberID            string
		wantMemberType          string
		wantRoleID              string
		wantOK                  bool
	}{
		{
			name:                    "valid",
			id:                      "domain-1/protected-resource-1/member-1/user/role-1",
			wantDomainID:            "domain-1",
			wantProtectedResourceID: "protected-resource-1",
			wantMemberID:            "member-1",
			wantMemberType:          "USER",
			wantRoleID:              "role-1",
			wantOK:                  true,
		},
		{
			name:   "missing part",
			id:     "domain-1/protected-resource-1/member-1/user",
			wantOK: false,
		},
		{
			name:   "extra part",
			id:     "domain-1/protected-resource-1/member-1/user/role-1/extra",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotProtectedResourceID, gotMemberID, gotMemberType, gotRoleID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotProtectedResourceID != tt.wantProtectedResourceID {
				t.Fatalf("protected resource ID = %q, want %q", gotProtectedResourceID, tt.wantProtectedResourceID)
			}
			if gotMemberID != tt.wantMemberID {
				t.Fatalf("member ID = %q, want %q", gotMemberID, tt.wantMemberID)
			}
			if gotMemberType != tt.wantMemberType {
				t.Fatalf("member type = %q, want %q", gotMemberType, tt.wantMemberType)
			}
			if gotRoleID != tt.wantRoleID {
				t.Fatalf("role ID = %q, want %q", gotRoleID, tt.wantRoleID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := ProtectedResourceMemberModel{
		MemberID:   types.StringValue("member-1"),
		MemberType: types.StringValue("group"),
		RoleID:     types.StringValue("role-1"),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"memberId":   "member-1",
		"memberType": "GROUP",
		"role":       "role-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestMatchesMembershipByID(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceMemberModel{
		ID:         types.StringValue("membership-id"),
		MemberID:   types.StringValue("different-member"),
		MemberType: types.StringValue("USER"),
		RoleID:     types.StringValue("different-role"),
	}

	if !matchesMembership(&model, map[string]interface{}{
		"id":         "membership-id",
		"memberId":   "member-id",
		"memberType": "GROUP",
		"roleId":     "role-id",
	}) {
		t.Fatal("matchesMembership returned false, want true by id")
	}
}

func TestMatchesMembershipByMemberTuple(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceMemberModel{
		MemberID:   types.StringValue("member-id"),
		MemberType: types.StringValue("user"),
		RoleID:     types.StringValue("role-id"),
	}

	if !matchesMembership(&model, map[string]interface{}{
		"id":         "membership-id",
		"memberId":   "member-id",
		"memberType": "USER",
		"roleId":     "role-id",
	}) {
		t.Fatal("matchesMembership returned false, want true by member tuple")
	}
}

func TestMatchesMembershipRejectsDifferentRole(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceMemberModel{
		MemberID:   types.StringValue("member-id"),
		MemberType: types.StringValue("USER"),
		RoleID:     types.StringValue("role-id"),
	}

	if matchesMembership(&model, map[string]interface{}{
		"memberId":   "member-id",
		"memberType": "USER",
		"roleId":     "other-role",
	}) {
		t.Fatal("matchesMembership returned true for different role")
	}
}

func TestReadIntoModelMapsProtectedResourceMembership(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceMemberModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":         "membership-id",
		"memberId":   "member-id",
		"memberType": "group",
		"roleId":     "role-id",
	})

	if model.ID.ValueString() != "membership-id" {
		t.Fatalf("id = %q, want membership-id", model.ID.ValueString())
	}
	if model.MemberID.ValueString() != "member-id" {
		t.Fatalf("member_id = %q, want member-id", model.MemberID.ValueString())
	}
	if model.MemberType.ValueString() != "GROUP" {
		t.Fatalf("member_type = %q, want GROUP", model.MemberType.ValueString())
	}
	if model.RoleID.ValueString() != "role-id" {
		t.Fatalf("role_id = %q, want role-id", model.RoleID.ValueString())
	}
}

func TestProtectedResourceMemberCRUDReadsManagedMembershipOnly(t *testing.T) {
	var bodies []map[string]interface{}
	var deletePaths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/members", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "membership-123"})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"memberships": []map[string]interface{}{
					{
						"id":         "unrelated",
						"memberId":   "user-123",
						"memberType": "USER",
						"roleId":     "other-role",
					},
					{
						"id":         "membership-123",
						"memberId":   "user-123",
						"memberType": "user",
						"roleId":     "role-123",
					},
				},
			})
		default:
			t.Fatalf("collection method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/members/membership-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s", r.Method)
		}
		deletePaths = append(deletePaths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ProtectedResourceMemberResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := protectedResourceMemberPlan(t, schemaResp.Schema, ProtectedResourceMemberModel{
		DomainID:            types.StringValue("domain-123"),
		ProtectedResourceID: types.StringValue("resource-123"),
		MemberID:            types.StringValue("user-123"),
		MemberType:          types.StringValue("user"),
		RoleID:              types.StringValue("role-123"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState ProtectedResourceMemberModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if got, want := createState.MemberType.ValueString(), "USER"; got != want {
		t.Fatalf("member type = %q, want %q", got, want)
	}
	if got, want := createState.RoleID.ValueString(), "role-123"; got != want {
		t.Fatalf("role = %q, want %q", got, want)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: readResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	wantBodies := []map[string]interface{}{{"memberId": "user-123", "memberType": "USER", "role": "role-123"}}
	if !reflect.DeepEqual(bodies, wantBodies) {
		t.Fatalf("bodies = %#v, want %#v", bodies, wantBodies)
	}
	wantDeletePaths := []string{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/members/membership-123"}
	if !reflect.DeepEqual(deletePaths, wantDeletePaths) {
		t.Fatalf("delete paths = %#v, want %#v", deletePaths, wantDeletePaths)
	}
}

func TestProtectedResourceMemberReadRemovesMissingMembershipAndDeleteIgnores404(t *testing.T) {
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/members", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("collection method = %s, want GET", r.Method)
		}
		methods = append(methods, "read")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"memberships": []map[string]interface{}{
				{
					"id":         "other-membership",
					"memberId":   "other-user",
					"memberType": "USER",
					"roleId":     "role-123",
				},
			},
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/members/missing-membership", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		methods = append(methods, "delete")
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ProtectedResourceMemberResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ProtectedResourceMemberModel{
		ID:                  types.StringValue("missing-membership"),
		DomainID:            types.StringValue("domain-123"),
		ProtectedResourceID: types.StringValue("resource-123"),
		MemberID:            types.StringValue("user-123"),
		MemberType:          types.StringValue("USER"),
		RoleID:              types.StringValue("role-123"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing membership to remove state, got %#v", readResp.State.Raw)
	}

	updateResp := &resource.UpdateResponse{}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected unsupported update diagnostic")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if !reflect.DeepEqual(methods, []string{"read", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
}

func TestProtectedResourceMemberReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/members", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			http.Error(w, "create failed", http.StatusInternalServerError)
		case http.MethodGet:
			http.Error(w, "read failed", http.StatusInternalServerError)
		default:
			t.Fatalf("collection method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/members/membership-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ProtectedResourceMemberResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := protectedResourceMemberPlan(t, schemaResp.Schema, ProtectedResourceMemberModel{
		DomainID:            types.StringValue("domain-123"),
		ProtectedResourceID: types.StringValue("resource-123"),
		MemberID:            types.StringValue("user-123"),
		MemberType:          types.StringValue("USER"),
		RoleID:              types.StringValue("role-123"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := protectedResourceMemberState(t, schemaResp.Schema, ProtectedResourceMemberModel{
		ID:                  types.StringValue("membership-123"),
		DomainID:            types.StringValue("domain-123"),
		ProtectedResourceID: types.StringValue("resource-123"),
		MemberID:            types.StringValue("user-123"),
		MemberType:          types.StringValue("USER"),
		RoleID:              types.StringValue("role-123"),
	})
	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestProtectedResourceMemberDeleteReportsInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ProtectedResourceMemberResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: invalidProtectedResourceMemberRaw()},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestProtectedResourceMemberCreateAndReadReportInvalidRequestData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ProtectedResourceMemberResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: invalidProtectedResourceMemberRaw()},
	}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected invalid create plan diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: invalidProtectedResourceMemberRaw()},
	}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected invalid read state diagnostics")
	}
}

func TestProtectedResourceMemberCreateReportsReadAfterCreateError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/members", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "membership-123"})
		case http.MethodGet:
			http.Error(w, "read after create failed", http.StatusInternalServerError)
		default:
			t.Fatalf("collection method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ProtectedResourceMemberResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := protectedResourceMemberPlan(t, schemaResp.Schema, ProtectedResourceMemberModel{
		DomainID:            types.StringValue("domain-123"),
		ProtectedResourceID: types.StringValue("resource-123"),
		MemberID:            types.StringValue("user-123"),
		MemberType:          types.StringValue("USER"),
		RoleID:              types.StringValue("role-123"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected read-after-create diagnostics")
	}
}

func TestProtectedResourceMemberImportStateSetsAttributes(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewProtectedResourceMemberResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := protectedResourceMemberState(t, schemaResp.Schema, ProtectedResourceMemberModel{
		ID:                  types.StringValue("membership-123"),
		DomainID:            types.StringValue("placeholder-domain"),
		ProtectedResourceID: types.StringValue("placeholder-resource"),
		MemberID:            types.StringValue("placeholder-member"),
		MemberType:          types.StringValue("USER"),
		RoleID:              types.StringValue("placeholder-role"),
	})
	importResp := &resource.ImportStateResponse{State: state}

	NewProtectedResourceMemberResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/resource-123/member-123/group/role-123",
	}, importResp)

	if importResp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", importResp.Diagnostics)
	}
	var imported ProtectedResourceMemberModel
	if diags := importResp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if imported.DomainID.ValueString() != "domain-123" ||
		imported.ProtectedResourceID.ValueString() != "resource-123" ||
		imported.MemberID.ValueString() != "member-123" ||
		imported.MemberType.ValueString() != "GROUP" ||
		imported.RoleID.ValueString() != "role-123" {
		t.Fatalf("imported = %#v", imported)
	}
}

func TestProtectedResourceMemberImportRejectsInvalidID(t *testing.T) {
	t.Parallel()

	var resp resource.ImportStateResponse
	NewProtectedResourceMemberResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain/resource/member/role",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import diagnostics")
	}
}

func protectedResourceMemberPlan(t *testing.T, schema resourceschema.Schema, model ProtectedResourceMemberModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func protectedResourceMemberState(t *testing.T, schema resourceschema.Schema, model ProtectedResourceMemberModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func invalidProtectedResourceMemberRaw() tftypes.Value {
	return tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":                    tftypes.String,
			"domain_id":             tftypes.Number,
			"protected_resource_id": tftypes.String,
			"member_id":             tftypes.String,
			"member_type":           tftypes.String,
			"role_id":               tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":                    tftypes.NewValue(tftypes.String, "membership-123"),
			"domain_id":             tftypes.NewValue(tftypes.Number, 123),
			"protected_resource_id": tftypes.NewValue(tftypes.String, "resource-123"),
			"member_id":             tftypes.NewValue(tftypes.String, "user-123"),
			"member_type":           tftypes.NewValue(tftypes.String, "USER"),
			"role_id":               tftypes.NewValue(tftypes.String, "role-123"),
		},
	)
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
