package orgsettings

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildPatchBodyIncludesIdentities(t *testing.T) {
	t.Parallel()

	model := OrgSettingsModel{
		Identities: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("idp-1"),
			types.StringValue("idp-2"),
		}),
	}

	got := buildPatchBody(model)
	want := map[string]interface{}{
		"identities": []string{"idp-1", "idp-2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildPatchBodyAllowsExplicitEmptyIdentities(t *testing.T) {
	t.Parallel()

	model := OrgSettingsModel{
		Identities: types.ListValueMust(types.StringType, []attr.Value{}),
	}

	got := buildPatchBody(model)
	want := map[string]interface{}{
		"identities": []string{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildPatchBodyOmitsUnknownOrNullIdentities(t *testing.T) {
	t.Parallel()

	for name, identities := range map[string]types.List{
		"null":    types.ListNull(types.StringType),
		"unknown": types.ListUnknown(types.StringType),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := buildPatchBody(OrgSettingsModel{Identities: identities})
			if len(got) != 0 {
				t.Fatalf("body = %#v, want empty", got)
			}
		})
	}
}

func TestReadIdentitiesMapsIdentityList(t *testing.T) {
	t.Parallel()

	model := OrgSettingsModel{}

	readIdentities(&model, map[string]interface{}{
		"identities": []interface{}{"idp-1", "idp-2"},
	})

	want := []string{"idp-1", "idp-2"}
	if got := listStrings(t, model.Identities); !reflect.DeepEqual(got, want) {
		t.Fatalf("identities = %#v, want %#v", got, want)
	}
}

func TestReadIdentitiesUsesEmptyListForMissingMalformedOrNilValue(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]interface{}{
		"missing": {},
		"nil": {
			"identities": nil,
		},
		"malformed": {
			"identities": "not-a-list",
		},
	}

	for name, result := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			model := OrgSettingsModel{
				Identities: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("old")}),
			}

			readIdentities(&model, result)

			if got := listStrings(t, model.Identities); len(got) != 0 {
				t.Fatalf("identities = %#v, want empty", got)
			}
		})
	}
}

func TestReadIdentitiesSkipsNonStringEntries(t *testing.T) {
	t.Parallel()

	model := OrgSettingsModel{}

	readIdentities(&model, map[string]interface{}{
		"identities": []interface{}{"idp-1", 42, "idp-2"},
	})

	want := []string{"idp-1", "idp-2"}
	if got := listStrings(t, model.Identities); !reflect.DeepEqual(got, want) {
		t.Fatalf("identities = %#v, want %#v", got, want)
	}
}

func listStrings(t *testing.T, list types.List) []string {
	t.Helper()

	values := make([]string, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		value, ok := element.(types.String)
		if !ok {
			t.Fatalf("element %T is not types.String", element)
		}
		values = append(values, value.ValueString())
	}
	return values
}
