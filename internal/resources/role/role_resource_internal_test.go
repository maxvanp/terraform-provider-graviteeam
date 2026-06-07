package role

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewRoleResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_role"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewRoleResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "description", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "assignable_type", false, true, false)
	assertListAttribute(t, resp.Schema.Attributes, "permissions", types.StringType)
	assertListAttribute(t, resp.Schema.Attributes, "oauth_scopes", types.StringType)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&RoleResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

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

func assertListAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, elemType attr.Type) {
	t.Helper()

	attr, ok := attrs[name].(schema.ListAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.ListAttribute", name, attrs[name])
	}
	if !attr.Optional || attr.Required || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want optional only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
	if !reflect.DeepEqual(attr.ElementType, elemType) {
		t.Fatalf("%s element type = %#v, want %#v", name, attr.ElementType, elemType)
	}
}
