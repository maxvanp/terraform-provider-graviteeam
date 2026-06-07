package group

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildUpdateBodyClearsRemovedGroupLists(t *testing.T) {
	plan := GroupModel{
		Name: types.StringValue("updated"),
	}
	state := GroupModel{
		Name:        types.StringValue("updated"),
		Description: types.StringValue("old description"),
		Members: []types.String{
			types.StringValue("user-id"),
		},
		Roles: []types.String{
			types.StringValue("role-id"),
		},
	}

	body := (&GroupResource{}).buildUpdateBody(plan, &state)

	if got := body["description"]; got != nil {
		t.Fatalf("description = %#v, want omitted when removed", got)
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

func TestReadIntoModelMapsGroupResponse(t *testing.T) {
	model := GroupModel{}

	(&GroupResource{}).readIntoModel(&model, map[string]interface{}{
		"id":          "group-id",
		"name":        "group-name",
		"description": "description",
		"members":     []interface{}{"user-1", "user-2"},
		"roles":       []interface{}{"role-1"},
	})

	if model.ID.ValueString() != "group-id" {
		t.Fatalf("id = %q, want group-id", model.ID.ValueString())
	}
	if model.Name.ValueString() != "group-name" {
		t.Fatalf("name = %q, want group-name", model.Name.ValueString())
	}
	if model.Description.ValueString() != "description" {
		t.Fatalf("description = %q, want description", model.Description.ValueString())
	}
	if len(model.Members) != 2 || model.Members[0].ValueString() != "user-1" || model.Members[1].ValueString() != "user-2" {
		t.Fatalf("members = %#v, want user-1,user-2", model.Members)
	}
	if len(model.Roles) != 1 || model.Roles[0].ValueString() != "role-1" {
		t.Fatalf("roles = %#v, want role-1", model.Roles)
	}
}
