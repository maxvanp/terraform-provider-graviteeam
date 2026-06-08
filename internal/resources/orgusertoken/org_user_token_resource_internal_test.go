package orgusertoken

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

func TestOrgUserTokenMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewOrgUserTokenResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_org_user_token" {
		t.Fatalf("type name = %q, want graviteeam_org_user_token", resp.TypeName)
	}
}

func TestOrgUserTokenSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgUserTokenResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"user_id", "name"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"id", "token_id", "token"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
	if attr := resp.Schema.Attributes["token"]; !attr.IsSensitive() {
		t.Fatal("token should be sensitive")
	}
}

func TestOrgUserTokenConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgUserTokenResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestOrgUserTokenConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgUserTokenResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestOrgUserTokenUpdateIsUnsupported(t *testing.T) {
	t.Parallel()

	var resp resource.UpdateResponse
	NewOrgUserTokenResource().Update(context.Background(), resource.UpdateRequest{}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostic")
	}
}

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := OrgUserTokenModel{
		Name: types.StringValue("automation-token"),
	}

	got := buildCreateBody(plan)
	want := map[string]interface{}{
		"name": "automation-token",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelMapsCreatedToken(t *testing.T) {
	t.Parallel()

	model := OrgUserTokenModel{
		UserID: types.StringValue("user-id"),
	}

	readIntoModel(&model, map[string]interface{}{
		"tokenId": "token-id",
		"name":    "automation-token",
		"token":   "secret-token-value",
	})

	if got, want := model.ID.ValueString(), "user-id/token-id"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.TokenID.ValueString(), "token-id"; got != want {
		t.Fatalf("token id = %q, want %q", got, want)
	}
	if got, want := model.Name.ValueString(), "automation-token"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Token.ValueString(), "secret-token-value"; got != want {
		t.Fatalf("token = %q, want %q", got, want)
	}
}

func TestReadIntoModelPreservesExistingTokenWhenAPIOmitsToken(t *testing.T) {
	t.Parallel()

	model := OrgUserTokenModel{
		UserID: types.StringValue("user-id"),
		Token:  types.StringValue("existing-secret"),
	}

	readIntoModel(&model, map[string]interface{}{
		"tokenId": "token-id",
		"name":    "automation-token",
	})

	if got, want := model.Token.ValueString(), "existing-secret"; got != want {
		t.Fatalf("token = %q, want %q", got, want)
	}
}

func TestReadIntoModelIgnoresEmptyToken(t *testing.T) {
	t.Parallel()

	model := OrgUserTokenModel{
		UserID: types.StringValue("user-id"),
		Token:  types.StringValue("existing-secret"),
	}

	readIntoModel(&model, map[string]interface{}{
		"tokenId": "token-id",
		"name":    "automation-token",
		"token":   "",
	})

	if got, want := model.Token.ValueString(), "existing-secret"; got != want {
		t.Fatalf("token = %q, want %q", got, want)
	}
}

func TestOrgUserTokenCRUDPreservesCreateOnlySecret(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	var createBodies []map[string]interface{}
	var deletePaths []string
	mux.HandleFunc("/management/organizations/DEFAULT/users/user-123/tokens", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			createBodies = append(createBodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"tokenId": "token-123",
				"name":    body["name"],
				"token":   "secret-token-value",
			})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"tokenId": "token-123", "name": "automation-token"},
				{"tokenId": "other-token", "name": "other"},
			})
		default:
			t.Fatalf("unexpected token collection method %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/user-123/tokens/token-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected token item method %s", r.Method)
		}
		deletePaths = append(deletePaths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgUserTokenResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgUserTokenPlan(t, schemaResp.Schema, OrgUserTokenModel{
		UserID: types.StringValue("user-123"),
		Name:   types.StringValue("automation-token"),
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
	var readState OrgUserTokenModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := readState.ID.ValueString(), "user-123/token-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := readState.Token.ValueString(), "secret-token-value"; got != want {
		t.Fatalf("token = %q, want preserved %q", got, want)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: readResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if want := []map[string]interface{}{{"name": "automation-token"}}; !reflect.DeepEqual(createBodies, want) {
		t.Fatalf("create bodies = %#v, want %#v", createBodies, want)
	}
	if want := []string{"/management/organizations/DEFAULT/users/user-123/tokens/token-123"}; !reflect.DeepEqual(deletePaths, want) {
		t.Fatalf("delete paths = %#v, want %#v", deletePaths, want)
	}
}

func TestOrgUserTokenReadRemovesMissingTokenAndDeleteIgnores404(t *testing.T) {
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/user-123/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("collection method = %s, want GET", r.Method)
		}
		methods = append(methods, "read")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"tokenId": "other-token", "name": "other"},
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/user-123/tokens/missing-token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		methods = append(methods, "delete")
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgUserTokenResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &OrgUserTokenModel{
		ID:      types.StringValue("user-123/missing-token"),
		UserID:  types.StringValue("user-123"),
		TokenID: types.StringValue("missing-token"),
		Name:    types.StringValue("automation-token"),
		Token:   types.StringValue("secret-token-value"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing token to remove state, got %#v", readResp.State.Raw)
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

func TestOrgUserTokenReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/user-123/tokens", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			http.Error(w, "create failed", http.StatusInternalServerError)
		case http.MethodGet:
			http.Error(w, "read failed", http.StatusInternalServerError)
		default:
			t.Fatalf("collection method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/user-123/tokens/token-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgUserTokenResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := orgUserTokenPlan(t, schemaResp.Schema, OrgUserTokenModel{
		UserID: types.StringValue("user-123"),
		Name:   types.StringValue("automation-token"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := orgUserTokenState(t, schemaResp.Schema, OrgUserTokenModel{
		ID:      types.StringValue("user-123/token-123"),
		UserID:  types.StringValue("user-123"),
		TokenID: types.StringValue("token-123"),
		Name:    types.StringValue("automation-token"),
		Token:   types.StringValue("secret-token-value"),
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

func TestOrgUserTokenImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&OrgUserTokenResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestOrgUserTokenImportStateSetsAttributes(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&OrgUserTokenResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: orgUserTokenState(t, schemaResp.Schema, OrgUserTokenModel{
		ID:      types.StringValue("old-user/old-token"),
		UserID:  types.StringValue("old-user"),
		TokenID: types.StringValue("old-token"),
		Name:    types.StringValue("automation-token"),
		Token:   types.StringValue("secret-token-value"),
	})}

	(&OrgUserTokenResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "user-123/token-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state OrgUserTokenModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get import state: %#v", diags)
	}
	if got, want := state.ID.ValueString(), "user-123/token-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := state.UserID.ValueString(), "user-123"; got != want {
		t.Fatalf("user_id = %q, want %q", got, want)
	}
	if got, want := state.TokenID.ValueString(), "token-123"; got != want {
		t.Fatalf("token_id = %q, want %q", got, want)
	}
}

func orgUserTokenPlan(t *testing.T, schema resourceschema.Schema, model OrgUserTokenModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func orgUserTokenState(t *testing.T, schema resourceschema.Schema, model OrgUserTokenModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
