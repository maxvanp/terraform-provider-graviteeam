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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

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

func userRolePlan(t *testing.T, schema resourceschema.Schema, model UserRoleModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
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
