package orgidentityprovider

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := OrgIdentityProviderModel{
		Name:          types.StringValue("org-idp"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{"users":[]}`),
		External:      types.BoolValue(false),
		DomainWhitelist: []types.String{
			types.StringValue("example.com"),
			types.StringValue("example.org"),
		},
	}

	got := buildCreateBody(plan)
	want := map[string]interface{}{
		"name":            "org-idp",
		"type":            "inline-am-idp",
		"configuration":   `{"users":[]}`,
		"external":        false,
		"domainWhitelist": []string{"example.com", "example.org"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildCreateBodyOmitsAbsentDomainWhitelist(t *testing.T) {
	t.Parallel()

	plan := OrgIdentityProviderModel{
		Name:          types.StringValue("org-idp"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{}`),
		External:      types.BoolValue(false),
	}

	got := buildCreateBody(plan)
	if _, ok := got["domainWhitelist"]; ok {
		t.Fatalf("domainWhitelist should be omitted, got %#v", got)
	}
}

func TestBuildUpdateBodyIncludesOptionalMappings(t *testing.T) {
	t.Parallel()

	groupMapper := mustConditionMap(t, map[string][]string{
		"{#profile['groups'].contains('external-admins')}": {"group-admin"},
		"{#profile['groups'].contains('external-users')}":  {"group-user", "group-admin"},
	})
	roleMapper := mustConditionMap(t, map[string][]string{
		"{#profile['groups'].contains('external-admin-role')}": {"role-admin"},
	})
	plan := OrgIdentityProviderModel{
		Name:          types.StringValue("org-idp"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{"users":[]}`),
		Mappers: map[string]types.String{
			"email":    types.StringValue("email"),
			"username": types.StringValue("username"),
		},
		DomainWhitelist: []types.String{types.StringValue("example.com")},
		GroupMapper:     groupMapper,
		RoleMapper:      roleMapper,
	}

	got := buildUpdateBody(plan, nil)

	if got["name"] != "org-idp" || got["type"] != "inline-am-idp" || got["configuration"] != `{"users":[]}` {
		t.Fatalf("base fields not mapped correctly: %#v", got)
	}
	if !reflect.DeepEqual(got["mappers"], map[string]string{"email": "email", "username": "username"}) {
		t.Fatalf("mappers = %#v", got["mappers"])
	}
	if !reflect.DeepEqual(got["domainWhitelist"], []string{"example.com"}) {
		t.Fatalf("domainWhitelist = %#v", got["domainWhitelist"])
	}

	groupAPI := got["groupMapper"].(map[string][]string)
	assertStringSet(t, groupAPI["group-admin"], []string{
		"{#profile['groups'].contains('external-admins')}",
		"{#profile['groups'].contains('external-users')}",
	})
	assertStringSet(t, groupAPI["group-user"], []string{
		"{#profile['groups'].contains('external-users')}",
	})

	roleAPI := got["roleMapper"].(map[string][]string)
	assertStringSet(t, roleAPI["role-admin"], []string{
		"{#profile['groups'].contains('external-admin-role')}",
	})
}

func TestBuildUpdateBodyClearsRemovedOptionalMappings(t *testing.T) {
	t.Parallel()

	state := OrgIdentityProviderModel{
		Mappers: map[string]types.String{
			"email": types.StringValue("email"),
		},
		DomainWhitelist: []types.String{types.StringValue("example.com")},
		GroupMapper: mustConditionMap(t, map[string][]string{
			"{true}": {"group-id"},
		}),
		RoleMapper: mustConditionMap(t, map[string][]string{
			"{true}": {"role-id"},
		}),
	}
	plan := OrgIdentityProviderModel{
		Name:          types.StringValue("org-idp"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{}`),
		GroupMapper:   types.MapNull(types.ListType{ElemType: types.StringType}),
		RoleMapper:    types.MapNull(types.ListType{ElemType: types.StringType}),
	}

	got := buildUpdateBody(plan, &state)

	if !reflect.DeepEqual(got["mappers"], map[string]string{}) {
		t.Fatalf("mappers clear = %#v", got["mappers"])
	}
	if !reflect.DeepEqual(got["domainWhitelist"], []string{}) {
		t.Fatalf("domainWhitelist clear = %#v", got["domainWhitelist"])
	}
	if !reflect.DeepEqual(got["groupMapper"], map[string][]string{}) {
		t.Fatalf("groupMapper clear = %#v", got["groupMapper"])
	}
	if !reflect.DeepEqual(got["roleMapper"], map[string][]string{}) {
		t.Fatalf("roleMapper clear = %#v", got["roleMapper"])
	}
}

func TestReadIntoModelMapsOrgIdentityProvider(t *testing.T) {
	t.Parallel()

	model := OrgIdentityProviderModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":            "idp-id",
		"name":          "org-idp",
		"type":          "inline-am-idp",
		"external":      true,
		"configuration": map[string]interface{}{"users": []interface{}{}},
		"mappers": map[string]interface{}{
			"email":    "email",
			"username": "username",
		},
		"domainWhitelist": []interface{}{"example.com"},
		"groupMapper": map[string]interface{}{
			"group-id": []interface{}{"{#profile['groups'].contains('external-users')}"},
		},
		"roleMapper": map[string]interface{}{
			"role-id": []interface{}{"{#profile['groups'].contains('external-admin-role')}"},
		},
	})

	if got, want := model.ID.ValueString(), "idp-id"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.Name.ValueString(), "org-idp"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "inline-am-idp"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if !model.External.ValueBool() {
		t.Fatalf("external should be true")
	}
	if got, want := model.Configuration.ValueString(), `{"users":[]}`; got != want {
		t.Fatalf("configuration = %q, want %q", got, want)
	}
	if got, want := model.Mappers["email"].ValueString(), "email"; got != want {
		t.Fatalf("mapper email = %q, want %q", got, want)
	}
	if got, want := stringValues(model.DomainWhitelist), []string{"example.com"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("domainWhitelist = %#v, want %#v", got, want)
	}
	assertListValue(t, model.GroupMapper.Elements()["{#profile['groups'].contains('external-users')}"], []string{"group-id"})
	assertListValue(t, model.RoleMapper.Elements()["{#profile['groups'].contains('external-admin-role')}"], []string{"role-id"})
}

func TestReadIntoModelClearsAbsentOptionalMappings(t *testing.T) {
	t.Parallel()

	model := OrgIdentityProviderModel{
		Mappers:         map[string]types.String{"email": types.StringValue("email")},
		DomainWhitelist: []types.String{types.StringValue("example.com")},
		GroupMapper: mustConditionMap(t, map[string][]string{
			"{true}": {"group-id"},
		}),
		RoleMapper: mustConditionMap(t, map[string][]string{
			"{true}": {"role-id"},
		}),
	}

	readIntoModel(&model, map[string]interface{}{})

	if model.Mappers != nil {
		t.Fatalf("mappers should be nil, got %#v", model.Mappers)
	}
	if model.DomainWhitelist != nil {
		t.Fatalf("domain whitelist should be nil, got %#v", model.DomainWhitelist)
	}
	if !model.GroupMapper.IsNull() {
		t.Fatalf("group mapper should be null")
	}
	if !model.RoleMapper.IsNull() {
		t.Fatalf("role mapper should be null")
	}
}

func TestInvertConditionMapperToAPI(t *testing.T) {
	t.Parallel()

	got := invertConditionMapperToAPI(mustConditionMap(t, map[string][]string{
		"{#profile['groups'].contains('external-admins')}": {"group-admin"},
		"{#profile['groups'].contains('external-users')}":  {"group-user", "group-admin"},
	}))

	assertStringSet(t, got["group-admin"], []string{
		"{#profile['groups'].contains('external-admins')}",
		"{#profile['groups'].contains('external-users')}",
	})
	assertStringSet(t, got["group-user"], []string{
		"{#profile['groups'].contains('external-users')}",
	})
}

func TestReadAPIConditionMapper(t *testing.T) {
	t.Parallel()

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

	assertListValue(t, got.Elements()["{#profile['groups'].contains('external-admins')}"], []string{"group-admin"})
	assertListValue(t, got.Elements()["{#profile['groups'].contains('external-users')}"], []string{"group-user"})
}

func mustConditionMap(t *testing.T, values map[string][]string) types.Map {
	t.Helper()

	listType := types.ListType{ElemType: types.StringType}
	elements := make(map[string]attr.Value, len(values))
	for condition, ids := range values {
		elements[condition] = mustStringList(t, ids...)
	}
	mapVal, diags := types.MapValue(listType, elements)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building mapper: %v", diags)
	}
	return mapVal
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

func stringValues(values []types.String) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.ValueString()
	}
	return result
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
