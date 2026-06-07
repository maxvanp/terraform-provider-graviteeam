package orggroupmembers

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOrgGroupMembersMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewOrgGroupMembersResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_org_group_members" {
		t.Fatalf("type name = %q, want graviteeam_org_group_members", resp.TypeName)
	}
}

func TestOrgGroupMembersSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgGroupMembersResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"group_id", "members"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if _, ok := resp.Schema.Attributes["domain_id"]; ok {
		t.Fatal("organization group members should not expose domain_id")
	}
	if !strings.Contains(resp.Schema.Description, "complete set") {
		t.Fatalf("schema description = %q, want complete-set ownership hint", resp.Schema.Description)
	}
}

func TestOrgGroupMembersConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgGroupMembersResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestValidImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		id     string
		wantOK bool
	}{
		{
			name:   "valid",
			id:     "group-1",
			wantOK: true,
		},
		{
			name:   "empty",
			id:     "",
			wantOK: false,
		},
		{
			name:   "blank",
			id:     "   ",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := validImportID(tt.id); got != tt.wantOK {
				t.Fatalf("validImportID(%q) = %t, want %t", tt.id, got, tt.wantOK)
			}
		})
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	model := OrgGroupMembersModel{
		GroupID: types.StringValue("group-1"),
		Members: []types.String{types.StringValue("old-user")},
	}

	readIntoModel(&model, []string{"user-1", "user-2"})

	gotMembers := stringValues(model.Members)
	wantMembers := []string{"user-1", "user-2"}
	if !reflect.DeepEqual(gotMembers, wantMembers) {
		t.Fatalf("members = %#v, want %#v", gotMembers, wantMembers)
	}
}

func TestDiffMembers(t *testing.T) {
	t.Parallel()

	toAdd, toRemove := diffMembers(
		[]types.String{types.StringValue("user-c"), types.StringValue("user-b"), types.StringValue("user-c")},
		[]types.String{types.StringValue("user-a"), types.StringValue("user-b"), types.StringValue("user-a")},
	)

	if want := []string{"user-c"}; !reflect.DeepEqual(toAdd, want) {
		t.Fatalf("toAdd = %#v, want %#v", toAdd, want)
	}
	if want := []string{"user-a"}; !reflect.DeepEqual(toRemove, want) {
		t.Fatalf("toRemove = %#v, want %#v", toRemove, want)
	}
}
