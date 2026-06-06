package extensiongrant

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildUpdateBodyClearsRemovedIdentityProvider(t *testing.T) {
	plan := ExtensionGrantModel{
		Name:          types.StringValue("updated"),
		Type:          types.StringValue("jwtbearer-am-extension-grant"),
		GrantType:     types.StringValue("urn:ietf:params:oauth:grant-type:jwt-bearer"),
		Configuration: types.StringValue("{}"),
		CreateUser:    types.BoolValue(false),
		UserExists:    types.BoolValue(true),
	}
	state := plan
	state.IdentityProvider = types.StringValue("idp-id")

	body := buildUpdateBody(plan, state)

	if got := body["identityProvider"]; got != "" {
		t.Fatalf("identityProvider = %#v, want empty string", got)
	}
}
