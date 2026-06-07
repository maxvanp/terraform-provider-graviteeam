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

func orgUserTokenPlan(t *testing.T, schema resourceschema.Schema, model OrgUserTokenModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
