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
