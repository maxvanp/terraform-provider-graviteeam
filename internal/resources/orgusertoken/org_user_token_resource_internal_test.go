package orgusertoken

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
