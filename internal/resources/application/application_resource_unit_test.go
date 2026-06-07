package application

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewApplicationResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_application"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewApplicationResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "type"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"description", "metadata_json", "settings_json", "identity_providers", "factors"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "client_id", "client_secret"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
	for _, name := range []string{"client_secret", "settings_json"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsSensitive() {
			t.Fatalf("attribute %q should be sensitive", name)
		}
	}
	for _, name := range []string{"identity_provider_rule", "oauth_settings", "mfa_settings"} {
		if _, ok := resp.Schema.Blocks[name]; !ok {
			t.Fatalf("missing schema block %q", name)
		}
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestApplicationBuildUpdateBodyMergesSettingsJSONWithTypedBlocks(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

func TestApplicationBuildBodiesIncludeMetadataJSON(t *testing.T) {
	t.Parallel()

	resource := &ApplicationResource{}
	plan := ApplicationModel{
		Name: types.StringValue("test-app"),
		Type: types.StringValue("WEB"),
		MetadataJSON: types.StringValue(`{
			"tenant": {
				"id": "tenant-a",
				"name": "Tenant A"
			}
		}`),
	}

	createBody, err := resource.buildCreateBody(plan)
	if err != nil {
		t.Fatalf("expected no create error, got %v", err)
	}
	updateBody, err := resource.buildUpdateBody(plan)
	if err != nil {
		t.Fatalf("expected no update error, got %v", err)
	}

	for name, body := range map[string]map[string]interface{}{
		"create": createBody,
		"update": updateBody,
	} {
		metadata, ok := body["metadata"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected %s metadata map, got %#v", name, body["metadata"])
		}
		tenant, ok := metadata["tenant"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected %s tenant metadata object, got %#v", name, metadata["tenant"])
		}
		if tenant["id"] != "tenant-a" || tenant["name"] != "Tenant A" {
			t.Fatalf("unexpected %s tenant metadata: %#v", name, tenant)
		}
	}
}

func TestApplicationBuildBodiesRejectInvalidMetadataJSON(t *testing.T) {
	t.Parallel()

	resource := &ApplicationResource{}
	for _, metadataJSON := range []string{`[]`, `null`} {
		plan := ApplicationModel{
			Name:         types.StringValue("test-app"),
			Type:         types.StringValue("WEB"),
			MetadataJSON: types.StringValue(metadataJSON),
		}

		if _, err := resource.buildCreateBody(plan); err == nil {
			t.Fatalf("expected create metadata_json %s to return an error", metadataJSON)
		}
		if _, err := resource.buildUpdateBody(plan); err == nil {
			t.Fatalf("expected update metadata_json %s to return an error", metadataJSON)
		}
	}
}

func TestApplicationReadIntoModelPreservesUnownedMetadata(t *testing.T) {
	t.Parallel()

	resource := &ApplicationResource{}
	model := &ApplicationModel{}

	resource.readIntoModel(model, map[string]interface{}{
		"metadata": map[string]interface{}{
			"tenant": map[string]interface{}{"id": "tenant-a"},
		},
	})

	if !model.MetadataJSON.IsNull() {
		t.Fatalf("expected metadata_json to remain null when not configured, got %s", model.MetadataJSON.ValueString())
	}
}

func TestApplicationReadIntoModelReadsOwnedMetadata(t *testing.T) {
	t.Parallel()

	resource := &ApplicationResource{}
	model := &ApplicationModel{
		MetadataJSON: types.StringValue(`{"tenant":{"id":"old"}}`),
	}

	resource.readIntoModel(model, map[string]interface{}{
		"metadata": map[string]interface{}{
			"tenant": map[string]interface{}{
				"id":   "tenant-a",
				"name": "Tenant A",
			},
		},
	})

	var metadata map[string]map[string]string
	if err := json.Unmarshal([]byte(model.MetadataJSON.ValueString()), &metadata); err != nil {
		t.Fatalf("expected metadata_json to be valid JSON, got %v", err)
	}
	if metadata["tenant"]["id"] != "tenant-a" || metadata["tenant"]["name"] != "Tenant A" {
		t.Fatalf("unexpected metadata_json value: %#v", metadata)
	}
}
