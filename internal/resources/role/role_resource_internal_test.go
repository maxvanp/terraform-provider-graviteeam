package role

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildUpdateBodyClearsRemovedRoleLists(t *testing.T) {
	plan := RoleModel{
		Name: types.StringValue("updated"),
	}
	state := RoleModel{
		Name:        types.StringValue("updated"),
		Description: types.StringValue("old description"),
		Permissions: []types.String{
			types.StringValue("DOMAIN_READ"),
		},
		OAuthScopes: []types.String{
			types.StringValue("scope-id"),
		},
	}

	body := (&RoleResource{}).buildUpdateBody(plan, &state)

	if got := body["description"]; got != nil {
		t.Fatalf("description = %#v, want omitted when removed", got)
	}
	perms, ok := body["permissions"].([]string)
	if !ok {
		t.Fatalf("permissions = %#v, want []string", body["permissions"])
	}
	if len(perms) != 0 {
		t.Fatalf("permissions length = %d, want 0", len(perms))
	}
	scopes, ok := body["oauthScopes"].([]string)
	if !ok {
		t.Fatalf("oauthScopes = %#v, want []string", body["oauthScopes"])
	}
	if len(scopes) != 0 {
		t.Fatalf("oauthScopes length = %d, want 0", len(scopes))
	}
}

func TestReadIntoModelMapsRoleResponse(t *testing.T) {
	model := RoleModel{}

	(&RoleResource{}).readIntoModel(&model, map[string]interface{}{
		"id":             "role-id",
		"name":           "role-name",
		"description":    "description",
		"assignableType": "domain",
		"permissions":    []interface{}{"DOMAIN_READ"},
		"oauthScopes":    []interface{}{"scope-id"},
	})

	if model.ID.ValueString() != "role-id" {
		t.Fatalf("id = %q, want role-id", model.ID.ValueString())
	}
	if model.Name.ValueString() != "role-name" {
		t.Fatalf("name = %q, want role-name", model.Name.ValueString())
	}
	if model.Description.ValueString() != "description" {
		t.Fatalf("description = %q, want description", model.Description.ValueString())
	}
	if model.AssignableType.ValueString() != "DOMAIN" {
		t.Fatalf("assignableType = %q, want DOMAIN", model.AssignableType.ValueString())
	}
	if len(model.Permissions) != 1 || model.Permissions[0].ValueString() != "DOMAIN_READ" {
		t.Fatalf("permissions = %#v, want DOMAIN_READ", model.Permissions)
	}
	if len(model.OAuthScopes) != 1 || model.OAuthScopes[0].ValueString() != "scope-id" {
		t.Fatalf("oauthScopes = %#v, want scope-id", model.OAuthScopes)
	}
}
