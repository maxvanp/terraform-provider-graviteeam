package application

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
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

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
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

func TestApplicationBuildUpdateBodyCoversSimpleIDPsMFAAndFullOAuth(t *testing.T) {
	t.Parallel()

	resource := &ApplicationResource{}
	plan := ApplicationModel{
		Name:              types.StringValue("test-app"),
		Description:       types.StringValue("full config"),
		IdentityProviders: []types.String{types.StringValue("idp-a"), types.StringValue("idp-b")},
		Factors:           []types.String{types.StringValue("factor-a"), types.StringValue("factor-b")},
		OAuthSettings: &OAuthSettingsModel{
			RedirectURIs:                []types.String{types.StringValue("https://app.example.test/callback")},
			PostLogoutRedirectURIs:      []types.String{types.StringValue("https://app.example.test/logout")},
			GrantTypes:                  []types.String{types.StringValue("authorization_code")},
			ResponseTypes:               []types.String{types.StringValue("code")},
			Scopes:                      []types.String{types.StringValue("openid"), types.StringValue("email")},
			AccessTokenValiditySeconds:  types.Int64Value(3600),
			RefreshTokenValiditySeconds: types.Int64Value(7200),
			IDTokenValiditySeconds:      types.Int64Value(1800),
		},
		MFASettings: &MFASettingsModel{
			Enrollment: types.StringValue("REQUIRED"),
			Challenge:  types.StringValue("OPTIONAL"),
		},
	}

	body, err := resource.buildUpdateBody(plan)
	if err != nil {
		t.Fatalf("build update body: %v", err)
	}

	idps, ok := body["identityProviders"].([]map[string]interface{})
	if !ok || len(idps) != 2 {
		t.Fatalf("identity providers = %#v", body["identityProviders"])
	}
	if idps[0]["identity"] != "idp-a" || idps[0]["priority"] != 0 || idps[1]["priority"] != 1 {
		t.Fatalf("identity provider priorities = %#v", idps)
	}
	if !reflect.DeepEqual(body["factors"], []string{"factor-a", "factor-b"}) {
		t.Fatalf("factors = %#v", body["factors"])
	}

	settings := body["settings"].(map[string]interface{})
	oauth := settings["oauth"].(map[string]interface{})
	if !reflect.DeepEqual(oauth["postLogoutRedirectUris"], []string{"https://app.example.test/logout"}) ||
		!reflect.DeepEqual(oauth["responseTypes"], []string{"code"}) ||
		oauth["refreshTokenValiditySeconds"] != int64(7200) ||
		oauth["idTokenValiditySeconds"] != int64(1800) {
		t.Fatalf("oauth settings = %#v", oauth)
	}
	scopes, ok := oauth["scopeSettings"].([]map[string]interface{})
	if !ok || len(scopes) != 2 || scopes[0]["scope"] != "openid" || scopes[0]["defaultScope"] != true {
		t.Fatalf("scope settings = %#v", oauth["scopeSettings"])
	}
	mfa := settings["mfa"].(map[string]interface{})
	factor := mfa["factor"].(map[string]interface{})
	if factor["defaultFactorId"] != "factor-a" {
		t.Fatalf("mfa factor = %#v", factor)
	}
	enroll := mfa["enroll"].(map[string]interface{})
	if enroll["type"] != "REQUIRED" || enroll["forceEnrollment"] != true {
		t.Fatalf("mfa enroll = %#v", enroll)
	}
	challenge := mfa["challenge"].(map[string]interface{})
	if challenge["type"] != "OPTIONAL" || challenge["active"] != true {
		t.Fatalf("mfa challenge = %#v", challenge)
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

func TestApplicationBuildCreateBodyRejectsInvalidSettingsRedirectURIs(t *testing.T) {
	t.Parallel()

	resource := &ApplicationResource{}
	for name, settingsJSON := range map[string]string{
		"not_list":   `{"oauth":{"redirectUris":"https://example.com/callback"}}`,
		"non_string": `{"oauth":{"redirectUris":["https://example.com/callback",42]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := resource.buildCreateBody(ApplicationModel{
				Name:         types.StringValue("test-app"),
				Type:         types.StringValue("WEB"),
				SettingsJSON: types.StringValue(settingsJSON),
			})
			if err == nil {
				t.Fatal("expected redirect URI diagnostics")
			}
		})
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

func TestApplicationReadIntoModelMapsSimpleListsRulesOAuthAndMFA(t *testing.T) {
	t.Parallel()

	resource := &ApplicationResource{}
	model := &ApplicationModel{
		OAuthSettings: &OAuthSettingsModel{},
		MFASettings: &MFASettingsModel{
			Enrollment: types.StringValue("OPTIONAL"),
			Challenge:  types.StringValue("OPTIONAL"),
		},
	}

	resource.readIntoModel(model, map[string]interface{}{
		"id":          "app-123",
		"name":        "app",
		"type":        "web",
		"description": "application",
		"identityProviders": []interface{}{
			map[string]interface{}{"identity": "idp-a", "priority": float64(0)},
			map[string]interface{}{"identity": "idp-b", "priority": int64(1)},
		},
		"factors": []interface{}{"factor-a", "factor-b"},
		"settings": map[string]interface{}{
			"oauth": map[string]interface{}{
				"redirectUris":                []interface{}{"https://app.example.test/callback"},
				"postLogoutRedirectUris":      []interface{}{"https://app.example.test/logout"},
				"grantTypes":                  []interface{}{"authorization_code"},
				"responseTypes":               []interface{}{"code"},
				"scopeSettings":               []interface{}{map[string]interface{}{"scope": "openid"}, map[string]interface{}{"scope": "email"}},
				"accessTokenValiditySeconds":  float64(3600),
				"refreshTokenValiditySeconds": int64(7200),
				"idTokenValiditySeconds":      "ignored",
			},
			"mfa": map[string]interface{}{
				"enroll":    map[string]interface{}{"type": "required"},
				"challenge": map[string]interface{}{"type": "conditional"},
			},
		},
	})

	if model.ID.ValueString() != "app-123" ||
		model.Type.ValueString() != "WEB" ||
		model.Description.ValueString() != "application" {
		t.Fatalf("basic model fields = %#v", model)
	}
	if got := []types.String{types.StringValue("idp-a"), types.StringValue("idp-b")}; !reflect.DeepEqual(model.IdentityProviders, got) {
		t.Fatalf("identity providers = %#v", model.IdentityProviders)
	}
	if got := []types.String{types.StringValue("factor-a"), types.StringValue("factor-b")}; !reflect.DeepEqual(model.Factors, got) {
		t.Fatalf("factors = %#v", model.Factors)
	}
	if model.OAuthSettings.AccessTokenValiditySeconds.ValueInt64() != 3600 ||
		model.OAuthSettings.RefreshTokenValiditySeconds.ValueInt64() != 7200 ||
		!model.OAuthSettings.IDTokenValiditySeconds.IsNull() {
		t.Fatalf("oauth validity fields = %#v", model.OAuthSettings)
	}
	if !reflect.DeepEqual(model.OAuthSettings.Scopes, []types.String{types.StringValue("openid"), types.StringValue("email")}) {
		t.Fatalf("oauth scopes = %#v", model.OAuthSettings.Scopes)
	}
	if model.MFASettings.Enrollment.ValueString() != "REQUIRED" ||
		model.MFASettings.Challenge.ValueString() != "CONDITIONAL" {
		t.Fatalf("mfa settings = %#v", model.MFASettings)
	}
}

func TestApplicationReadIntoModelMapsIdentityProviderRules(t *testing.T) {
	t.Parallel()

	resource := &ApplicationResource{}
	model := &ApplicationModel{
		IdentityProviderRules: []IdentityProviderRuleModel{
			{Identity: types.StringValue("planned"), Priority: types.Int64Value(99)},
		},
	}

	resource.readIntoModel(model, map[string]interface{}{
		"identityProviders": []interface{}{
			map[string]interface{}{
				"identity":      "idp-a",
				"selectionRule": "{#context.attributes['tenant'] == 'a'}",
				"priority":      float64(3),
			},
			map[string]interface{}{
				"identity": "idp-b",
			},
		},
	})

	if len(model.IdentityProviderRules) != 2 {
		t.Fatalf("rules = %#v", model.IdentityProviderRules)
	}
	if model.IdentityProviderRules[0].Identity.ValueString() != "idp-a" ||
		model.IdentityProviderRules[0].SelectionRule.ValueString() == "" ||
		model.IdentityProviderRules[0].Priority.ValueInt64() != 3 {
		t.Fatalf("first rule = %#v", model.IdentityProviderRules[0])
	}
	if model.IdentityProviderRules[1].Identity.ValueString() != "idp-b" ||
		model.IdentityProviderRules[1].SelectionRule.ValueString() != "" ||
		model.IdentityProviderRules[1].Priority.ValueInt64() != 1 {
		t.Fatalf("second rule = %#v", model.IdentityProviderRules[1])
	}
	if model.IdentityProviders != nil {
		t.Fatalf("simple identity providers should be cleared, got %#v", model.IdentityProviders)
	}
}

func TestMergeApplicationUpdateBodyPreservesUnmanagedCurrentFields(t *testing.T) {
	t.Parallel()

	current := map[string]interface{}{
		"id":                  "ignored",
		"certificate":         "certificate-1",
		"description":         "old description",
		"enabled":             true,
		"requiredPermissions": []interface{}{"APPLICATION_READ"},
		"settings": map[string]interface{}{
			"advanced": map[string]interface{}{"skipConsent": true},
		},
		"template":   true,
		"unexpected": "must-not-leak",
	}
	update := map[string]interface{}{
		"name":        "updated",
		"description": "new description",
		"settings": map[string]interface{}{
			"oauth": map[string]interface{}{"redirectUris": []interface{}{"https://app.example.test/callback"}},
		},
	}

	got := mergeApplicationUpdateBody(current, update)

	for _, field := range []string{"certificate", "enabled", "requiredPermissions", "template"} {
		if !reflect.DeepEqual(got[field], current[field]) {
			t.Fatalf("%s = %#v, want preserved %#v", field, got[field], current[field])
		}
	}
	if got["name"] != "updated" || got["description"] != "new description" {
		t.Fatalf("planned fields not applied: %#v", got)
	}
	if !reflect.DeepEqual(got["settings"], update["settings"]) {
		t.Fatalf("settings = %#v, want update settings", got["settings"])
	}
	if _, ok := got["unexpected"]; ok {
		t.Fatalf("unexpected field leaked into update body: %#v", got)
	}
	if _, ok := got["id"]; ok {
		t.Fatalf("id leaked into update body: %#v", got)
	}
}

func TestApplicationCRUDPreservesSecretAndMergesUpdatePayloads(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(applicationResponse("app-123", map[string]interface{}{
			"name":        body["name"],
			"type":        body["type"],
			"description": body["description"],
			"settings": map[string]interface{}{
				"oauth": map[string]interface{}{
					"clientId":     "client-123",
					"clientSecret": "clear-secret",
				},
			},
		}))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(applicationResponse("app-123", map[string]interface{}{
				"name":                "app",
				"type":                "web",
				"description":         "created",
				"certificate":         "certificate-1",
				"enabled":             true,
				"requiredPermissions": []interface{}{"APPLICATION_READ"},
				"template":            true,
				"metadata":            map[string]interface{}{"owner": map[string]interface{}{"team": "iam"}},
				"identityProviders": []interface{}{
					map[string]interface{}{"identity": "idp-current", "priority": float64(0)},
				},
				"factors": []interface{}{"factor-current"},
				"settings": map[string]interface{}{
					"advanced": map[string]interface{}{"skipConsent": true},
					"oauth": map[string]interface{}{
						"clientId":     "client-123",
						"clientSecret": "********",
						"redirectUris": []interface{}{"https://app.example.test/callback"},
					},
				},
			}))
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(applicationResponse("app-123", body))
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/type", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("type method = %s, want PUT", r.Method)
		}
		methods = append(methods, "type")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode type body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(applicationResponse("app-123", map[string]interface{}{
			"name":        "app-updated",
			"type":        body["type"],
			"description": "updated",
			"settings": map[string]interface{}{
				"oauth": map[string]interface{}{
					"clientId":     "client-123",
					"clientSecret": "********",
				},
			},
		}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := applicationPlan(t, schemaResp.Schema, ApplicationModel{
		DomainID:     types.StringValue("domain-123"),
		Name:         types.StringValue("app"),
		Type:         types.StringValue("WEB"),
		Description:  types.StringValue("created"),
		MetadataJSON: types.StringValue(`{"owner":{"team":"iam"}}`),
		SettingsJSON: types.StringValue(`{"oauth":{"redirectUris":["https://app.example.test/callback"]}}`),
		IdentityProviderRules: []IdentityProviderRuleModel{
			{
				Identity:      types.StringValue("idp-1"),
				SelectionRule: types.StringValue("{#context.attributes['tenant'] == 'a'}"),
				Priority:      types.Int64Value(0),
			},
		},
		Factors: []types.String{types.StringValue("factor-1")},
		OAuthSettings: &OAuthSettingsModel{
			RedirectURIs: []types.String{types.StringValue("https://app.example.test/callback")},
			GrantTypes:   []types.String{types.StringValue("authorization_code")},
		},
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState ApplicationModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if createState.ClientSecret.ValueString() != "clear-secret" {
		t.Fatalf("client_secret after create = %q", createState.ClientSecret.ValueString())
	}

	updatePlan := applicationPlan(t, schemaResp.Schema, ApplicationModel{
		DomainID:     types.StringValue("domain-123"),
		Name:         types.StringValue("app-updated"),
		Type:         types.StringValue("BROWSER"),
		Description:  types.StringValue("updated"),
		MetadataJSON: types.StringValue(`{"owner":{"team":"platform"}}`),
		SettingsJSON: types.StringValue(`{"oauth":{"redirectUris":["https://app.example.test/updated"]}}`),
		IdentityProviders: []types.String{
			types.StringValue("idp-2"),
		},
		Factors: []types.String{types.StringValue("factor-2")},
		OAuthSettings: &OAuthSettingsModel{
			RedirectURIs: []types.String{types.StringValue("https://app.example.test/updated")},
			GrantTypes:   []types.String{types.StringValue("authorization_code")},
		},
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: createResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	var updateState ApplicationModel
	if diags := updateResp.State.Get(context.Background(), &updateState); diags.HasError() {
		t.Fatalf("get update state: %#v", diags)
	}
	if updateState.ClientSecret.ValueString() != "clear-secret" {
		t.Fatalf("client_secret after update = %q", updateState.ClientSecret.ValueString())
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if !reflect.DeepEqual(methods, []string{"create", "read", "update", "read", "update", "type", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if got := bodies[0]["type"]; got != "WEB" {
		t.Fatalf("create type = %#v", got)
	}
	if _, ok := bodies[1]["type"]; ok {
		t.Fatalf("post-create update body should not include type: %#v", bodies[1])
	}
	for _, bodyIndex := range []int{1, 2} {
		body := bodies[bodyIndex]
		for _, field := range []string{"certificate", "enabled", "requiredPermissions", "template"} {
			if _, ok := body[field]; !ok {
				t.Fatalf("update body %d missing preserved field %q: %#v", bodyIndex, field, body)
			}
		}
	}
	if got := bodies[2]["name"]; got != "app-updated" {
		t.Fatalf("update name = %#v", got)
	}
	if got := bodies[3]["type"]; got != "BROWSER" {
		t.Fatalf("type update body = %#v", bodies[3])
	}
}

func TestApplicationReadRemovesMissingApplicationAndDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ApplicationModel{
		ID:           types.StringValue("app-123"),
		DomainID:     types.StringValue("domain-123"),
		Name:         types.StringValue("app"),
		Type:         types.StringValue("WEB"),
		Description:  types.StringValue("application"),
		ClientID:     types.StringValue("client-123"),
		ClientSecret: types.StringValue("clear-secret"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing application to remove state, got %#v", readResp.State.Raw)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestApplicationCRUDReportsRemoteErrors(t *testing.T) {
	tests := map[string]struct {
		createStatus int
		itemStatus   int
		typeStatus   int
		action       func(context.Context, *ApplicationResource, tfsdk.Plan, tfsdk.State, resourceschema.Schema) bool
	}{
		"create": {
			createStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ApplicationResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"post_create_read": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ApplicationResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"read": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ApplicationResource, _ tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
				r.Read(ctx, resource.ReadRequest{State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"update_read_before": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ApplicationResource, plan tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"update_type": {
			typeStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ApplicationResource, plan tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"delete": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ApplicationResource, _ tfsdk.Plan, state tfsdk.State, _ resourceschema.Schema) bool {
				resp := &resource.DeleteResponse{}
				r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("collection method = %s, want POST", r.Method)
				}
				if tc.createStatus != 0 {
					http.Error(w, "remote error", tc.createStatus)
					return
				}
				_ = json.NewEncoder(w).Encode(applicationResponse("app-123", map[string]interface{}{
					"name": "app",
					"type": "WEB",
					"settings": map[string]interface{}{
						"oauth": map[string]interface{}{
							"clientId":     "client-123",
							"clientSecret": "clear-secret",
						},
					},
				}))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123", func(w http.ResponseWriter, r *http.Request) {
				if tc.itemStatus != 0 {
					http.Error(w, "remote error", tc.itemStatus)
					return
				}
				switch r.Method {
				case http.MethodGet, http.MethodPut:
					_ = json.NewEncoder(w).Encode(applicationResponse("app-123", map[string]interface{}{
						"name":        "app",
						"type":        "WEB",
						"description": "application",
						"settings": map[string]interface{}{
							"oauth": map[string]interface{}{"clientId": "client-123", "clientSecret": "********"},
						},
					}))
				case http.MethodDelete:
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Fatalf("item method = %s", r.Method)
				}
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/type", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Fatalf("type method = %s, want PUT", r.Method)
				}
				if tc.typeStatus != 0 {
					http.Error(w, "remote error", tc.typeStatus)
					return
				}
				_ = json.NewEncoder(w).Encode(applicationResponse("app-123", map[string]interface{}{
					"name": "app",
					"type": "BROWSER",
				}))
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &ApplicationResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			planModel := ApplicationModel{
				ID:           types.StringValue("app-123"),
				DomainID:     types.StringValue("domain-123"),
				Name:         types.StringValue("app-updated"),
				Type:         types.StringValue("BROWSER"),
				Description:  types.StringValue("updated"),
				ClientID:     types.StringValue("client-123"),
				ClientSecret: types.StringValue("clear-secret"),
			}
			stateModel := planModel
			stateModel.Name = types.StringValue("app")
			stateModel.Type = types.StringValue("WEB")
			stateModel.Description = types.StringValue("application")
			plan := applicationPlan(t, schemaResp.Schema, planModel)
			state := applicationState(t, schemaResp.Schema, stateModel)

			if !tc.action(context.Background(), resourceUnderTest, plan, state, schemaResp.Schema) {
				t.Fatal("expected diagnostics")
			}
		})
	}
}

func TestApplicationImportStateRejectsInvalidID(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewApplicationResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	importResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	NewApplicationResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "application-only",
	}, importResp)

	if !importResp.Diagnostics.HasError() {
		t.Fatal("expected invalid import diagnostics")
	}
}

func applicationResponse(id string, body map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"id": id,
	}
	for key, value := range body {
		result[key] = value
	}
	return result
}

func applicationState(t *testing.T, schema resourceschema.Schema, model ApplicationModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func applicationPlan(t *testing.T, schema resourceschema.Schema, model ApplicationModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
