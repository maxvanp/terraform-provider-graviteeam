package application

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplicationBuildUpdateBodyMergesSettingsJSONWithTypedBlocks(t *testing.T) {
	resource := &ApplicationResource{}
	plan := ApplicationModel{
		Name: types.StringValue("test-app"),
		SettingsJSON: types.StringValue(`{
			"advanced": {
				"skipConsent": true
			},
			"oauth": {
				"forcePKCE": true,
				"tokenEndpointAuthMethod": "client_secret_post",
				"tokenCustomClaims": [
					{
						"claimName": "tenant",
						"claimValue": "{#context.attributes['tenant']}",
						"tokenType": "ACCESS_TOKEN"
					}
				]
			}
		}`),
		OAuthSettings: &OAuthSettingsModel{
			RedirectURIs:               []types.String{types.StringValue("https://example.com/callback")},
			GrantTypes:                 []types.String{types.StringValue("authorization_code")},
			AccessTokenValiditySeconds: types.Int64Value(3600),
		},
	}

	body, err := resource.buildUpdateBody(plan)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	settings, ok := body["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected settings in body, got %#v", body["settings"])
	}

	advanced, ok := settings["advanced"].(map[string]interface{})
	if !ok || advanced["skipConsent"] != true {
		t.Fatalf("expected advanced settings from settings_json, got %#v", settings["advanced"])
	}

	oauth, ok := settings["oauth"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected oauth settings, got %#v", settings["oauth"])
	}
	if oauth["forcePKCE"] != true {
		t.Fatalf("expected forcePKCE from settings_json, got %#v", oauth["forcePKCE"])
	}
	if oauth["tokenEndpointAuthMethod"] != "client_secret_post" {
		t.Fatalf("expected tokenEndpointAuthMethod from settings_json, got %#v", oauth["tokenEndpointAuthMethod"])
	}
	if oauth["accessTokenValiditySeconds"] != int64(3600) {
		t.Fatalf("expected typed access token validity to be merged, got %#v", oauth["accessTokenValiditySeconds"])
	}
	if !reflect.DeepEqual(oauth["redirectUris"], []string{"https://example.com/callback"}) {
		t.Fatalf("expected typed redirect URIs to be merged, got %#v", oauth["redirectUris"])
	}
	if !reflect.DeepEqual(oauth["grantTypes"], []string{"authorization_code"}) {
		t.Fatalf("expected typed grant types to be merged, got %#v", oauth["grantTypes"])
	}
}

func TestApplicationBuildUpdateBodyRejectsInvalidSettingsJSON(t *testing.T) {
	resource := &ApplicationResource{}
	for _, settingsJSON := range []string{`[]`, `null`} {
		plan := ApplicationModel{
			Name:         types.StringValue("test-app"),
			SettingsJSON: types.StringValue(settingsJSON),
		}

		_, err := resource.buildUpdateBody(plan)
		if err == nil {
			t.Fatalf("expected settings_json %s to return an error", settingsJSON)
		}
	}
}

func TestApplicationBuildCreateBodyUsesSettingsJSONRedirectURIs(t *testing.T) {
	resource := &ApplicationResource{}
	plan := ApplicationModel{
		Name: types.StringValue("test-app"),
		Type: types.StringValue("WEB"),
		SettingsJSON: types.StringValue(`{
			"oauth": {
				"redirectUris": ["https://example.com/callback"]
			}
		}`),
	}

	body, err := resource.buildCreateBody(plan)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !reflect.DeepEqual(body["redirectUris"], []string{"https://example.com/callback"}) {
		t.Fatalf("expected redirect URIs from settings_json, got %#v", body["redirectUris"])
	}
}
