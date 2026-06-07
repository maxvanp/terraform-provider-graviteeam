package orggroup

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewOrgGroupResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_group"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgGroupResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["name"]; attr == nil || !attr.IsRequired() {
		t.Fatalf("name should be required")
	}
	for _, name := range []string{"description", "members", "roles"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; attr == nil || !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgGroupResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildBodyAppliesPlannedOrganizationGroupCollections(t *testing.T) {
	t.Parallel()

	model := OrgGroupModel{
		Name:        types.StringValue("admins"),
		Description: types.StringValue("Admin group"),
		Members:     []types.String{types.StringValue("user-1"), types.StringValue("user-2")},
		Roles:       []types.String{types.StringValue("role-1")},
	}

	got := buildBody(model, OrgGroupModel{})
	want := map[string]interface{}{
		"name":        "admins",
		"description": "Admin group",
		"members":     []string{"user-1", "user-2"},
		"roles":       []string{"role-1"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyClearsDescriptionMembersAndRoles(t *testing.T) {
	t.Parallel()

	model := OrgGroupModel{
		Name:        types.StringValue("admins"),
		Description: types.StringNull(),
		Members:     nil,
		Roles:       nil,
	}
	state := OrgGroupModel{
		Description: types.StringValue("old"),
		Members:     []types.String{types.StringValue("user-1")},
		Roles:       []types.String{types.StringValue("role-1")},
	}

	got := buildBody(model, state)

	if got["description"] != "" {
		t.Fatalf("description = %#v, want empty string", got["description"])
	}
	if !reflect.DeepEqual(got["members"], []string{}) {
		t.Fatalf("members = %#v, want clear list", got["members"])
	}
	if !reflect.DeepEqual(got["roles"], []string{}) {
		t.Fatalf("roles = %#v, want clear list", got["roles"])
	}
}

func TestReadIntoModelMapsOrganizationGroupFields(t *testing.T) {
	t.Parallel()

	model := OrgGroupModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":          "group-1",
		"name":        "admins",
		"description": "Admin group",
		"members":     []interface{}{"user-1", "user-2"},
		"roles":       []interface{}{"role-1"},
	})

	if model.ID.ValueString() != "group-1" ||
		model.Name.ValueString() != "admins" ||
		model.Description.ValueString() != "Admin group" {
		t.Fatalf("model fields not mapped: %#v", model)
	}
	if got := stringValues(model.Members); !reflect.DeepEqual(got, []string{"user-1", "user-2"}) {
		t.Fatalf("members = %#v", got)
	}
	if got := stringValues(model.Roles); !reflect.DeepEqual(got, []string{"role-1"}) {
		t.Fatalf("roles = %#v", got)
	}
}

func TestReadIntoModelClearsEmptyOrganizationGroupCollections(t *testing.T) {
	t.Parallel()

	model := OrgGroupModel{
		Description: types.StringValue("old"),
		Members:     []types.String{types.StringValue("user-1")},
		Roles:       []types.String{types.StringValue("role-1")},
	}

	readIntoModel(&model, map[string]interface{}{
		"description": "",
		"members":     []interface{}{},
		"roles":       []interface{}{},
	})

	if !model.Description.IsNull() {
		t.Fatalf("description = %#v, want null", model.Description)
	}
	if model.Members != nil || model.Roles != nil {
		t.Fatalf("collections = members %#v roles %#v, want nil", model.Members, model.Roles)
	}
}

func TestInterfaceStringsIgnoresNonStringValuesAsZeroValue(t *testing.T) {
	t.Parallel()

	got := interfaceStrings([]interface{}{"user-1", 42})

	if got[0].ValueString() != "user-1" || !got[1].IsNull() {
		t.Fatalf("converted values = %#v", got)
	}
}
