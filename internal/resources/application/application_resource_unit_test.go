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

func applicationResponse(id string, body map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"id": id,
	}
	for key, value := range body {
		result[key] = value
	}
	return result
}

func applicationPlan(t *testing.T, schema resourceschema.Schema, model ApplicationModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
