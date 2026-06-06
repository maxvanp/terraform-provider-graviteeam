package identityprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestInvertConditionMapperToAPI(t *testing.T) {
	listType := types.ListType{ElemType: types.StringType}
	mapper, diags := types.MapValue(listType, map[string]attr.Value{
		"{#profile['groups'].contains('external-admins')}": mustStringList(t, "group-admin"),
		"{#profile['groups'].contains('external-users')}":  mustStringList(t, "group-user", "group-admin"),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building mapper: %v", diags)
	}

	got := invertConditionMapperToAPI(mapper)

	assertStringSet(t, got["group-admin"], []string{
		"{#profile['groups'].contains('external-admins')}",
		"{#profile['groups'].contains('external-users')}",
	})
	assertStringSet(t, got["group-user"], []string{
		"{#profile['groups'].contains('external-users')}",
	})
}

func TestReadAPIConditionMapper(t *testing.T) {
	listType := types.ListType{ElemType: types.StringType}
	got := readAPIConditionMapper(map[string]interface{}{
		"groupMapper": map[string]interface{}{
			"group-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
			"group-user":  []interface{}{"{#profile['groups'].contains('external-users')}"},
		},
	}, "groupMapper", listType)

	if got.IsNull() || got.IsUnknown() {
		t.Fatalf("expected concrete mapper, got %v", got)
	}

	elements := got.Elements()
	if len(elements) != 2 {
		t.Fatalf("expected 2 mapper entries, got %d", len(elements))
	}
	assertListValue(t, elements["{#profile['groups'].contains('external-admins')}"], []string{"group-admin"})
	assertListValue(t, elements["{#profile['groups'].contains('external-users')}"], []string{"group-user"})
}

func mustStringList(t *testing.T, values ...string) types.List {
	t.Helper()

	listValues := make([]attr.Value, len(values))
	for i, value := range values {
		listValues[i] = types.StringValue(value)
	}
	list, diags := types.ListValue(types.StringType, listValues)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building list: %v", diags)
	}
	return list
}

func assertStringSet(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	seen := make(map[string]bool, len(got))
	for _, value := range got {
		seen[value] = true
	}
	for _, value := range want {
		if !seen[value] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func assertListValue(t *testing.T, got attr.Value, want []string) {
	t.Helper()

	list, ok := got.(types.List)
	if !ok {
		t.Fatalf("expected types.List, got %T", got)
	}
	values := make([]string, 0, len(list.Elements()))
	for _, elem := range list.Elements() {
		value, ok := elem.(types.String)
		if !ok {
			t.Fatalf("expected types.String, got %T", elem)
		}
		values = append(values, value.ValueString())
	}
	assertStringSet(t, values, want)
}
