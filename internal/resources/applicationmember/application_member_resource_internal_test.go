package applicationmember

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
	NewApplicationMemberResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_application_member"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewApplicationMemberResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "application_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "member_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "member_type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "role_id", true, false, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationMemberResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationMemberResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestMatchesMembershipByID(t *testing.T) {
	model := ApplicationMemberModel{
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
	model := ApplicationMemberModel{
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
	model := ApplicationMemberModel{
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

func TestReadIntoModelMapsApplicationMembership(t *testing.T) {
	model := ApplicationMemberModel{}

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

func TestApplicationMemberCRUDReadsMembershipFromCollection(t *testing.T) {
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
	var deletePaths []string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members", func(w http.ResponseWriter, r *http.Request) {
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
						"id":         "membership-123",
						"memberId":   "user-123",
						"memberType": "user",
						"roleId":     "role-123",
					},
				},
			})
		default:
			t.Fatalf("unexpected application member collection method %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members/membership-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected application member item method %s", r.Method)
		}
		deletePaths = append(deletePaths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationMemberResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := applicationMemberPlan(t, schemaResp.Schema, ApplicationMemberModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		MemberID:      types.StringValue("user-123"),
		MemberType:    types.StringValue("user"),
		RoleID:        types.StringValue("role-123"),
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
	var readState ApplicationMemberModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := readState.MemberType.ValueString(), "USER"; got != want {
		t.Fatalf("member type = %q, want %q", got, want)
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
	wantDeletePaths := []string{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members/membership-123"}
	if !reflect.DeepEqual(deletePaths, wantDeletePaths) {
		t.Fatalf("delete paths = %#v, want %#v", deletePaths, wantDeletePaths)
	}
}

func TestApplicationMemberReadRemovesMissingMembershipAndDeleteIgnores404(t *testing.T) {
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members/missing-membership", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		methods = append(methods, "delete")
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationMemberResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ApplicationMemberModel{
		ID:            types.StringValue("missing-membership"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		MemberID:      types.StringValue("user-123"),
		MemberType:    types.StringValue("USER"),
		RoleID:        types.StringValue("role-123"),
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

func TestApplicationMemberReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			http.Error(w, "create failed", http.StatusInternalServerError)
		case http.MethodGet:
			http.Error(w, "read failed", http.StatusInternalServerError)
		default:
			t.Fatalf("collection method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members/membership-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationMemberResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := applicationMemberPlan(t, schemaResp.Schema, ApplicationMemberModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		MemberID:      types.StringValue("user-123"),
		MemberType:    types.StringValue("USER"),
		RoleID:        types.StringValue("role-123"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := applicationMemberState(t, schemaResp.Schema, ApplicationMemberModel{
		ID:            types.StringValue("membership-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		MemberID:      types.StringValue("user-123"),
		MemberType:    types.StringValue("USER"),
		RoleID:        types.StringValue("role-123"),
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

func TestApplicationMemberDeleteReportsInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ApplicationMemberResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"domain_id":      tftypes.Number,
			"application_id": tftypes.String,
			"member_id":      tftypes.String,
			"member_type":    tftypes.String,
			"role_id":        tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":             tftypes.NewValue(tftypes.String, "membership-123"),
			"domain_id":      tftypes.NewValue(tftypes.Number, 123),
			"application_id": tftypes.NewValue(tftypes.String, "app-123"),
			"member_id":      tftypes.NewValue(tftypes.String, "user-123"),
			"member_type":    tftypes.NewValue(tftypes.String, "USER"),
			"role_id":        tftypes.NewValue(tftypes.String, "role-123"),
		},
	)

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestApplicationMemberCreateReportsReadAfterCreateError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/members", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &ApplicationMemberResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := applicationMemberPlan(t, schemaResp.Schema, ApplicationMemberModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		MemberID:      types.StringValue("user-123"),
		MemberType:    types.StringValue("USER"),
		RoleID:        types.StringValue("role-123"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected read-after-create diagnostics")
	}
}

func TestApplicationMemberImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&ApplicationMemberResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain/app/member/type",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestApplicationMemberImportStateSetsAttributes(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&ApplicationMemberResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: applicationMemberState(t, schemaResp.Schema, ApplicationMemberModel{
		ID:            types.StringValue("membership-123"),
		DomainID:      types.StringValue("old-domain"),
		ApplicationID: types.StringValue("old-app"),
		MemberID:      types.StringValue("old-member"),
		MemberType:    types.StringValue("USER"),
		RoleID:        types.StringValue("old-role"),
	})}

	(&ApplicationMemberResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/app-123/user-123/group/role-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state ApplicationMemberModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get import state: %#v", diags)
	}
	if got, want := state.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain_id = %q, want %q", got, want)
	}
	if got, want := state.ApplicationID.ValueString(), "app-123"; got != want {
		t.Fatalf("application_id = %q, want %q", got, want)
	}
	if got, want := state.MemberID.ValueString(), "user-123"; got != want {
		t.Fatalf("member_id = %q, want %q", got, want)
	}
	if got, want := state.MemberType.ValueString(), "GROUP"; got != want {
		t.Fatalf("member_type = %q, want %q", got, want)
	}
	if got, want := state.RoleID.ValueString(), "role-123"; got != want {
		t.Fatalf("role_id = %q, want %q", got, want)
	}
}

func applicationMemberPlan(t *testing.T, schema resourceschema.Schema, model ApplicationMemberModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func applicationMemberState(t *testing.T, schema resourceschema.Schema, model ApplicationMemberModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
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
