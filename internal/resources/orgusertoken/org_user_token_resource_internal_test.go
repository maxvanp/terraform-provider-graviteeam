package orgusertoken

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
