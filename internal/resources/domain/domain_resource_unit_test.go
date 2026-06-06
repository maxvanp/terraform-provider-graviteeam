package domain

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDomainBuildUpdateBodyMergesCurrentPatchFields(t *testing.T) {
	resource := &DomainResource{}
	plan := DomainModel{
		Name:        types.StringValue("updated"),
		Description: types.StringValue("managed description"),
		Enabled:     types.BoolValue(true),
		DataPlaneID: types.StringValue("default"),
		OIDC: &OIDCModel{
			AllowLocalhostRedirectURI:        types.BoolValue(true),
			AllowHTTPSchemeRedirectURI:       types.BoolValue(false),
			AllowWildcardRedirectURI:         types.BoolValue(true),
			DynamicClientRegistrationEnabled: types.BoolValue(true),
		},
		LoginSettings: &LoginSettingsModel{
			RegisterEnabled:        types.BoolValue(true),
			ForgotPasswordEnabled:  types.BoolValue(false),
			IdentifierFirstEnabled: types.BoolValue(true),
		},
	}
	current := map[string]interface{}{
		"id":          "domain-id",
		"name":        "current",
		"description": "current description",
		"enabled":     false,
		"tags":        []interface{}{"team-a"},
		"saml":        map[string]interface{}{"enabled": true},
		"uma":         map[string]interface{}{"enabled": true},
		"oidc": map[string]interface{}{
			"clientRegistrationSettings": map[string]interface{}{
				"allowLocalhostRedirectUri":          false,
				"allowHttpSchemeRedirectUri":         true,
				"allowWildCardRedirectUri":           false,
				"isDynamicClientRegistrationEnabled": false,
				"preservedNestedFlag":                true,
			},
			"preservedOIDCSetting": true,
		},
		"loginSettings": map[string]interface{}{
			"registerEnabled":        false,
			"forgotPasswordEnabled":  true,
			"identifierFirstEnabled": false,
			"preservedLoginSetting":  true,
		},
	}

	body := resource.buildUpdateBody(plan, current)

	if body["id"] != nil {
		t.Fatalf("id should not be sent in PatchDomain body: %#v", body["id"])
	}
	if body["name"] != "updated" {
		t.Fatalf("expected managed name to override current value, got %#v", body["name"])
	}
	if body["description"] != "managed description" {
		t.Fatalf("expected managed description to override current value, got %#v", body["description"])
	}
	if body["enabled"] != true {
		t.Fatalf("expected managed enabled to override current value, got %#v", body["enabled"])
	}
	if _, ok := body["tags"]; !ok {
		t.Fatal("expected tags to be preserved")
	}
	if _, ok := body["saml"]; !ok {
		t.Fatal("expected saml settings to be preserved")
	}
	if _, ok := body["uma"]; !ok {
		t.Fatal("expected uma settings to be preserved")
	}

	oidc := body["oidc"].(map[string]interface{})
	if oidc["preservedOIDCSetting"] != true {
		t.Fatalf("expected unrelated oidc settings to be preserved, got %#v", oidc)
	}
	crs := oidc["clientRegistrationSettings"].(map[string]interface{})
	if crs["allowLocalhostRedirectUri"] != true || crs["allowWildCardRedirectUri"] != true {
		t.Fatalf("expected managed oidc settings to override current values, got %#v", crs)
	}
	if crs["preservedNestedFlag"] != true {
		t.Fatalf("expected unrelated nested oidc setting to be preserved, got %#v", crs)
	}

	loginSettings := body["loginSettings"].(map[string]interface{})
	if loginSettings["registerEnabled"] != true || loginSettings["identifierFirstEnabled"] != true {
		t.Fatalf("expected managed login settings to override current values, got %#v", loginSettings)
	}
	if loginSettings["preservedLoginSetting"] != true {
		t.Fatalf("expected unrelated login setting to be preserved, got %#v", loginSettings)
	}
}
