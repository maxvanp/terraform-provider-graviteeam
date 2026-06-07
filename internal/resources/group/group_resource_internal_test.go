package group

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
	NewGroupResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_group"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewGroupResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "description", false, true, false)
	assertListAttribute(t, resp.Schema.Attributes, "members", types.StringType)
	assertListAttribute(t, resp.Schema.Attributes, "roles", types.StringType)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&GroupResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

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
