package orgrole

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
