package groupmembers

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGroupMembersMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewGroupMembersResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_group_members" {
		t.Fatalf("type name = %q, want graviteeam_group_members", resp.TypeName)
	}
}

func TestGroupMembersSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewGroupMembersResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "group_id", "members"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if !strings.Contains(resp.Schema.Description, "complete membership list") {
		t.Fatalf("schema description = %q, want complete-list ownership hint", resp.Schema.Description)
	}
}

func TestGroupMembersConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GroupMembersResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		id           string
		wantDomainID string
		wantGroupID  string
		wantOK       bool
	}{
		{
			name:         "valid",
			id:           "domain-1/group-1",
			wantDomainID: "domain-1",
			wantGroupID:  "group-1",
			wantOK:       true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
		{
			name:   "too many segments",
			id:     "domain-1/group-1/extra",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotGroupID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotGroupID != tt.wantGroupID {
				t.Fatalf("group ID = %q, want %q", gotGroupID, tt.wantGroupID)
			}
		})
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	model := GroupMembersModel{
		DomainID: types.StringValue("domain-1"),
		GroupID:  types.StringValue("group-1"),
		Members:  []types.String{types.StringValue("old-user")},
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
