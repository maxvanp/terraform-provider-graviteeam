package domainmember

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		id             string
		wantDomainID   string
		wantMemberID   string
		wantMemberType string
		wantRoleID     string
		wantOK         bool
	}{
		{
			name:           "valid",
			id:             "domain-1/member-1/user/role-1",
			wantDomainID:   "domain-1",
			wantMemberID:   "member-1",
			wantMemberType: "USER",
			wantRoleID:     "role-1",
			wantOK:         true,
		},
		{
			name:   "missing part",
			id:     "domain-1/member-1/user",
			wantOK: false,
		},
		{
			name:   "extra part",
			id:     "domain-1/member-1/user/role-1/extra",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotMemberID, gotMemberType, gotRoleID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotMemberID != tt.wantMemberID {
				t.Fatalf("member ID = %q, want %q", gotMemberID, tt.wantMemberID)
			}
			if gotMemberType != tt.wantMemberType {
				t.Fatalf("member type = %q, want %q", gotMemberType, tt.wantMemberType)
			}
			if gotRoleID != tt.wantRoleID {
				t.Fatalf("role ID = %q, want %q", gotRoleID, tt.wantRoleID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := DomainMemberModel{
		MemberID:   types.StringValue("member-1"),
		MemberType: types.StringValue("group"),
		RoleID:     types.StringValue("role-1"),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"memberId":   "member-1",
		"memberType": "GROUP",
		"role":       "role-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestMatchesMembershipByID(t *testing.T) {
	t.Parallel()

	model := DomainMemberModel{
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
	t.Parallel()

	model := DomainMemberModel{
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
	t.Parallel()

	model := DomainMemberModel{
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

func TestReadIntoModelMapsDomainMembership(t *testing.T) {
	t.Parallel()

	model := DomainMemberModel{}

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
