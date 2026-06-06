package orgrole

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildUpdateBodyClearsRemovedOptionalFields(t *testing.T) {
	plan := OrgRoleModel{
		Name: types.StringValue("updated"),
	}
	state := OrgRoleModel{
		Name:        types.StringValue("updated"),
		Description: types.StringValue("old description"),
		Permissions: []types.String{
			types.StringValue("DOMAIN_READ"),
		},
	}

	body := buildUpdateBody(plan, state)

	if got := body["description"]; got != "" {
		t.Fatalf("description = %#v, want empty string", got)
	}

	perms, ok := body["permissions"].([]string)
	if !ok {
		t.Fatalf("permissions = %#v, want []string", body["permissions"])
	}
	if len(perms) != 0 {
		t.Fatalf("permissions length = %d, want 0", len(perms))
	}
}
