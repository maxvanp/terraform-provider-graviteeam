package orgrole

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
	NewOrgRoleResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_role"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgRoleResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "description", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "assignable_type", true, false, false)
	assertListAttribute(t, resp.Schema.Attributes, "permissions", types.StringType)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgRoleResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestBuildUpdateBodyAppliesPlannedFieldsAndPermissions(t *testing.T) {
	t.Parallel()

	plan := OrgRoleModel{
		Name:        types.StringValue("platform admin"),
		Description: types.StringValue("Admin role"),
		Permissions: []types.String{
			types.StringValue("organization_role_read"),
			types.StringValue("organization_role_update"),
		},
	}

	got := buildUpdateBody(plan, OrgRoleModel{})
	want := map[string]interface{}{
		"name":        "platform admin",
		"description": "Admin role",
		"permissions": []string{"organization_role_read", "organization_role_update"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyClearsDescriptionAndPermissions(t *testing.T) {
	t.Parallel()

	plan := OrgRoleModel{
		Name:        types.StringValue("platform admin"),
		Description: types.StringNull(),
		Permissions: nil,
	}
	state := OrgRoleModel{
		Description: types.StringValue("old"),
		Permissions: []types.String{
			types.StringValue("organization_role_read"),
		},
	}

	got := buildUpdateBody(plan, state)

	if got["description"] != "" {
		t.Fatalf("description = %#v, want empty string", got["description"])
	}
	if !reflect.DeepEqual(got["permissions"], []string{}) {
		t.Fatalf("permissions = %#v, want clear list", got["permissions"])
	}
}

func TestReadIntoModelMapsOrganizationRoleFields(t *testing.T) {
	t.Parallel()

	model := OrgRoleModel{}

	readIntoModel(&model, map[string]interface{}{
		"name":           "platform admin",
		"description":    "Admin role",
		"assignableType": "organization",
		"permissions": []interface{}{
			"organization_role_read",
			"organization_role_update",
		},
	})

	if model.Name.ValueString() != "platform admin" ||
		model.Description.ValueString() != "Admin role" ||
		model.AssignableType.ValueString() != "ORGANIZATION" {
		t.Fatalf("model fields not mapped: %#v", model)
	}
	if got := []string{model.Permissions[0].ValueString(), model.Permissions[1].ValueString()}; !reflect.DeepEqual(got, []string{"organization_role_read", "organization_role_update"}) {
		t.Fatalf("permissions = %#v", got)
	}
}

func TestReadIntoModelClearsEmptyOrganizationRoleOptionals(t *testing.T) {
	t.Parallel()

	model := OrgRoleModel{
		Description: types.StringValue("old"),
		Permissions: []types.String{
			types.StringValue("organization_role_read"),
		},
	}

	readIntoModel(&model, map[string]interface{}{
		"description": "",
		"permissions": []interface{}{},
	})

	if !model.Description.IsNull() {
		t.Fatalf("description = %#v, want null", model.Description)
	}
	if model.Permissions != nil {
		t.Fatalf("permissions = %#v, want nil", model.Permissions)
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
