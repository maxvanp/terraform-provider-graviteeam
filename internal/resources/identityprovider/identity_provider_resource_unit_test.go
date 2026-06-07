package identityprovider

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
	NewIdentityProviderResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_identity_provider"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewIdentityProviderResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", true, false, false)
	assertBoolAttribute(t, resp.Schema.Attributes, "external", false, true, true)
	assertStringAttribute(t, resp.Schema.Attributes, "configuration", true, false, false)
	assertMapAttribute(t, resp.Schema.Attributes, "mappers", types.StringType)
	assertListAttribute(t, resp.Schema.Attributes, "domain_whitelist", types.StringType)
	assertStringAttribute(t, resp.Schema.Attributes, "password_policy_id", false, true, false)
	assertMapAttribute(t, resp.Schema.Attributes, "group_mapper", types.ListType{ElemType: types.StringType})
	assertMapAttribute(t, resp.Schema.Attributes, "role_mapper", types.ListType{ElemType: types.StringType})
}

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

func TestBuildUpdateBodyClearsRemovedIdentityProviderCollections(t *testing.T) {
	t.Parallel()

	listType := types.ListType{ElemType: types.StringType}
	stateMapper, diags := types.MapValue(listType, map[string]attr.Value{
		"{#profile['groups'].contains('legacy')}": mustStringList(t, "group-old"),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building mapper: %v", diags)
	}

	plan := IdentityProviderModel{
		Name:          types.StringValue("inline"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{"users":[]}`),
		GroupMapper:   types.MapNull(listType),
		RoleMapper:    types.MapNull(listType),
	}
	state := IdentityProviderModel{
		Mappers:         map[string]types.String{"email": types.StringValue("mail")},
		DomainWhitelist: []types.String{types.StringValue("example.com")},
		GroupMapper:     stateMapper,
		RoleMapper:      stateMapper,
	}

	got := (&IdentityProviderResource{}).buildUpdateBody(plan, &state)

	if !reflect.DeepEqual(got["mappers"], map[string]string{}) {
		t.Fatalf("mappers = %#v, want empty map", got["mappers"])
	}
	if !reflect.DeepEqual(got["domainWhitelist"], []string{}) {
		t.Fatalf("domainWhitelist = %#v, want empty list", got["domainWhitelist"])
	}
	if !reflect.DeepEqual(got["groupMapper"], map[string][]string{}) {
		t.Fatalf("groupMapper = %#v, want empty map", got["groupMapper"])
	}
	if !reflect.DeepEqual(got["roleMapper"], map[string][]string{}) {
		t.Fatalf("roleMapper = %#v, want empty map", got["roleMapper"])
	}
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

func assertBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.BoolAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}

func assertMapAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, elemType attr.Type) {
	t.Helper()

	attr, ok := attrs[name].(schema.MapAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.MapAttribute", name, attrs[name])
	}
	if !attr.Optional || attr.Required || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want optional only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
	if !reflect.DeepEqual(attr.ElementType, elemType) {
		t.Fatalf("%s element type = %#v, want %#v", name, attr.ElementType, elemType)
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
