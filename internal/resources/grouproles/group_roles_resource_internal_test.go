package grouproles

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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

	model := GroupRolesModel{
		DomainID: types.StringValue("domain-1"),
		GroupID:  types.StringValue("group-1"),
		Roles:    []types.String{types.StringValue("old-role")},
	}

	readIntoModel(&model, []interface{}{
		map[string]interface{}{"id": "role-1"},
		map[string]interface{}{"name": "missing-id"},
		map[string]interface{}{"id": 123},
		"not-a-map",
		map[string]interface{}{"id": "role-2"},
	})

	gotRoles := stringValues(model.Roles)
	wantRoles := []string{"role-1", "role-2"}
	if !reflect.DeepEqual(gotRoles, wantRoles) {
		t.Fatalf("roles = %#v, want %#v", gotRoles, wantRoles)
	}
}

func TestDiffRoles(t *testing.T) {
	t.Parallel()

	toAdd, toRemove := diffRoles(
		[]types.String{types.StringValue("role-c"), types.StringValue("role-b"), types.StringValue("role-c")},
		[]types.String{types.StringValue("role-a"), types.StringValue("role-b"), types.StringValue("role-a")},
	)

	if want := []string{"role-c"}; !reflect.DeepEqual(toAdd, want) {
		t.Fatalf("toAdd = %#v, want %#v", toAdd, want)
	}
	if want := []string{"role-a"}; !reflect.DeepEqual(toRemove, want) {
		t.Fatalf("toRemove = %#v, want %#v", toRemove, want)
	}
}
