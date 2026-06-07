package extensiongrant

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildUpdateBodyIncludesIdentityProviderWhenPlanned(t *testing.T) {
	t.Parallel()

	plan := ExtensionGrantModel{
		Name:             types.StringValue("jwt bearer"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{"issuer":"test"}`),
		IdentityProvider: types.StringValue("idp-1"),
		CreateUser:       types.BoolValue(true),
		UserExists:       types.BoolValue(false),
	}

	got := buildUpdateBody(plan, ExtensionGrantModel{})
	want := map[string]interface{}{
		"name":             "jwt bearer",
		"type":             "jwtbearer-am-extension-grant",
		"grantType":        "urn:ietf:params:oauth:grant-type:jwt-bearer",
		"configuration":    `{"issuer":"test"}`,
		"identityProvider": "idp-1",
		"createUser":       true,
		"userExists":       false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyClearsExistingIdentityProviderWhenPlanNull(t *testing.T) {
	t.Parallel()

	plan := ExtensionGrantModel{
		Name:             types.StringValue("jwt bearer"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{}`),
		IdentityProvider: types.StringNull(),
		CreateUser:       types.BoolValue(false),
		UserExists:       types.BoolValue(true),
	}
	state := ExtensionGrantModel{
		IdentityProvider: types.StringValue("idp-1"),
	}

	got := buildUpdateBody(plan, state)

	if got["identityProvider"] != "" {
		t.Fatalf("identityProvider = %#v, want clear string", got["identityProvider"])
	}
}

func TestBuildUpdateBodyOmitsIdentityProviderWhenPlanAndStateNull(t *testing.T) {
	t.Parallel()

	plan := ExtensionGrantModel{
		Name:             types.StringValue("jwt bearer"),
		Type:             types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:        types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration:    types.StringValue(`{}`),
		IdentityProvider: types.StringNull(),
		CreateUser:       types.BoolValue(false),
		UserExists:       types.BoolValue(false),
	}

	got := buildUpdateBody(plan, ExtensionGrantModel{IdentityProvider: types.StringNull()})

	if _, ok := got["identityProvider"]; ok {
		t.Fatalf("identityProvider should be omitted: %#v", got)
	}
}

func TestReadIntoModelMapsExtensionGrantFields(t *testing.T) {
	t.Parallel()

	model := ExtensionGrantModel{}

	readIntoModel(&model, map[string]interface{}{
		"name":             "jwt bearer",
		"type":             "jwtbearer-am-extension-grant",
		"grantType":        "urn:ietf:params:oauth:grant-type:jwt-bearer",
		"configuration":    `{"issuer":"test"}`,
		"identityProvider": "idp-1",
		"createUser":       true,
		"userExists":       false,
	})

	if model.Name.ValueString() != "jwt bearer" ||
		model.Type.ValueString() != "jwtbearer-am-extension-grant" ||
		model.GrantType.ValueString() != "urn:ietf:params:oauth:grant-type:jwt-bearer" ||
		model.Configuration.ValueString() != `{"issuer":"test"}` ||
		model.IdentityProvider.ValueString() != "idp-1" ||
		!model.CreateUser.ValueBool() ||
		model.UserExists.ValueBool() {
		t.Fatalf("model = %#v", model)
	}
}

func TestReadIntoModelClearsEmptyIdentityProvider(t *testing.T) {
	t.Parallel()

	model := ExtensionGrantModel{
		IdentityProvider: types.StringValue("idp-1"),
	}

	readIntoModel(&model, map[string]interface{}{
		"identityProvider": "",
	})

	if !model.IdentityProvider.IsNull() {
		t.Fatalf("identityProvider = %#v, want null", model.IdentityProvider)
	}
}

func TestReadIntoModelKeepsNullIdentityProviderWhenMissing(t *testing.T) {
	t.Parallel()

	model := ExtensionGrantModel{
		IdentityProvider: types.StringNull(),
	}

	readIntoModel(&model, map[string]interface{}{})

	if !model.IdentityProvider.IsNull() {
		t.Fatalf("identityProvider = %#v, want null", model.IdentityProvider)
	}
}
