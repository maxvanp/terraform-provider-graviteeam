package orggroup

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildBodyClearsRemovedOptionalFields(t *testing.T) {
	plan := OrgGroupModel{
		Name: types.StringValue("updated"),
	}
	state := OrgGroupModel{
		Name:        types.StringValue("updated"),
		Description: types.StringValue("old description"),
		Members: []types.String{
			types.StringValue("user-id"),
		},
		Roles: []types.String{
			types.StringValue("role-id"),
		},
	}

	body := buildBody(plan, state)

	if got := body["description"]; got != "" {
		t.Fatalf("description = %#v, want empty string", got)
	}

	members, ok := body["members"].([]string)
	if !ok {
		t.Fatalf("members = %#v, want []string", body["members"])
	}
	if len(members) != 0 {
		t.Fatalf("members length = %d, want 0", len(members))
	}

	roles, ok := body["roles"].([]string)
	if !ok {
		t.Fatalf("roles = %#v, want []string", body["roles"])
	}
	if len(roles) != 0 {
		t.Fatalf("roles length = %d, want 0", len(roles))
	}
}
