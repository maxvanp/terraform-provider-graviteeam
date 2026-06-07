package applicationmember

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewApplicationMemberResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_application_member"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewApplicationMemberResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "application_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "member_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "member_type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "role_id", true, false, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationMemberResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestMatchesMembershipByID(t *testing.T) {
	model := ApplicationMemberModel{
		ID:         types.StringValue("membership-id"),
		MemberID:   types.StringValue("different-member"),
		MemberType: types.StringValue("USER"),
		RoleID:     types.StringValue("different-role"),
	}

	if !matchesMembership(&model, map[string]interface{}{
		"id":         "membership-id",
		"memberId":   "member-id",
		"memberType": "GROUP",
		"roleId":     "role-id",
	}) {
		t.Fatal("matchesMembership returned false, want true by id")
	}
}

func TestMatchesMembershipByMemberTuple(t *testing.T) {
	model := ApplicationMemberModel{
		MemberID:   types.StringValue("member-id"),
		MemberType: types.StringValue("user"),
		RoleID:     types.StringValue("role-id"),
	}

	if !matchesMembership(&model, map[string]interface{}{
		"id":         "membership-id",
		"memberId":   "member-id",
		"memberType": "USER",
		"roleId":     "role-id",
	}) {
		t.Fatal("matchesMembership returned false, want true by member tuple")
	}
}

func TestMatchesMembershipRejectsDifferentRole(t *testing.T) {
	model := ApplicationMemberModel{
		MemberID:   types.StringValue("member-id"),
		MemberType: types.StringValue("USER"),
		RoleID:     types.StringValue("role-id"),
	}

	if matchesMembership(&model, map[string]interface{}{
		"memberId":   "member-id",
		"memberType": "USER",
		"roleId":     "other-role",
	}) {
		t.Fatal("matchesMembership returned true for different role")
	}
}

func TestReadIntoModelMapsApplicationMembership(t *testing.T) {
	model := ApplicationMemberModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":         "membership-id",
		"memberId":   "member-id",
		"memberType": "group",
		"roleId":     "role-id",
	})

	if model.ID.ValueString() != "membership-id" {
		t.Fatalf("id = %q, want membership-id", model.ID.ValueString())
	}
	if model.MemberID.ValueString() != "member-id" {
		t.Fatalf("member_id = %q, want member-id", model.MemberID.ValueString())
	}
	if model.MemberType.ValueString() != "GROUP" {
		t.Fatalf("member_type = %q, want GROUP", model.MemberType.ValueString())
	}
	if model.RoleID.ValueString() != "role-id" {
		t.Fatalf("role_id = %q, want role-id", model.RoleID.ValueString())
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
