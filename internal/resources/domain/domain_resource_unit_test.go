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
		SettingsJSON: types.StringValue(`{
			"tags": ["team-b"],
			"saml": {
				"enabled": false
			},
			"oidc": {
				"securityProfileSettings": {
					"enablePlainFapi": true
				}
			}
		}`),
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

	body, err := resource.buildUpdateBody(plan, current)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

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
	if tags, ok := body["tags"].([]interface{}); !ok || tags[0] != "team-b" {
		t.Fatalf("expected settings_json tags to override current tags, got %#v", body["tags"])
	}
	if saml, ok := body["saml"].(map[string]interface{}); !ok || saml["enabled"] != false {
		t.Fatalf("expected settings_json saml settings to override current saml, got %#v", body["saml"])
	}
	if _, ok := body["uma"]; !ok {
		t.Fatal("expected uma settings to be preserved")
	}

	oidc := body["oidc"].(map[string]interface{})
	if oidc["preservedOIDCSetting"] != true {
		t.Fatalf("expected unrelated oidc settings to be preserved, got %#v", oidc)
	}
	securityProfile := oidc["securityProfileSettings"].(map[string]interface{})
	if securityProfile["enablePlainFapi"] != true {
		t.Fatalf("expected settings_json oidc settings to be merged, got %#v", securityProfile)
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

func TestDomainBuildUpdateBodyRejectsInvalidSettingsJSON(t *testing.T) {
	resource := &DomainResource{}
	for _, settingsJSON := range []string{`[]`, `null`} {
		plan := DomainModel{
			Name:         types.StringValue("test-domain"),
			Enabled:      types.BoolValue(false),
			DataPlaneID:  types.StringValue("default"),
			SettingsJSON: types.StringValue(settingsJSON),
		}

		if _, err := resource.buildUpdateBody(plan, nil); err == nil {
			t.Fatalf("expected settings_json %s to return an error", settingsJSON)
		}
	}
}
