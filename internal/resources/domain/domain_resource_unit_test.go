package domain

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
	NewDomainResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_domain"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewDomainResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["name"]; attr == nil || !attr.IsRequired() {
		t.Fatalf("name should be required")
	}
	for _, name := range []string{"description", "enabled", "data_plane_id", "settings_json"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "enabled", "data_plane_id", "default_idp_id"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
	for _, name := range []string{"oidc", "login_settings"} {
		if _, ok := resp.Schema.Blocks[name]; !ok {
			t.Fatalf("missing schema block %q", name)
		}
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&DomainResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestDomainBuildUpdateBodyMergesCurrentPatchFields(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

func TestDomainCRUDUsesPatchMergeAndDerivesDefaultIDP(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(domainResponse("domain-123", map[string]interface{}{
			"name":        body["name"],
			"description": body["description"],
			"dataPlaneId": body["dataPlaneId"],
			"enabled":     false,
		}))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(domainResponse("domain-123", map[string]interface{}{
				"name":                "domain",
				"description":         "created",
				"dataPlaneId":         "default",
				"enabled":             true,
				"tags":                []interface{}{"team-a"},
				"uma":                 map[string]interface{}{"enabled": true},
				"requiredPermissions": []interface{}{"DOMAIN_READ"},
				"oidc": map[string]interface{}{
					"clientRegistrationSettings": map[string]interface{}{
						"allowLocalhostRedirectUri":          true,
						"allowHttpSchemeRedirectUri":         false,
						"allowWildCardRedirectUri":           true,
						"isDynamicClientRegistrationEnabled": true,
						"preservedNestedFlag":                true,
					},
					"securityProfileSettings": map[string]interface{}{"enablePlainFapi": true},
				},
				"loginSettings": map[string]interface{}{
					"registerEnabled":        true,
					"forgotPasswordEnabled":  false,
					"identifierFirstEnabled": true,
					"preservedLoginSetting":  true,
				},
			}))
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(domainResponse("domain-123", body))
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &DomainResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := domainPlan(t, schemaResp.Schema, DomainModel{
		Name:         types.StringValue("domain"),
		Description:  types.StringValue("created"),
		Enabled:      types.BoolValue(true),
		DataPlaneID:  types.StringValue("default"),
		SettingsJSON: types.StringValue(`{"tags":["team-a"],"uma":{"enabled":true}}`),
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
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState DomainModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if createState.DefaultIdpID.ValueString() != "default-idp-domain-123" {
		t.Fatalf("default_idp_id = %q", createState.DefaultIdpID.ValueString())
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}

	updatePlan := domainPlan(t, schemaResp.Schema, DomainModel{
		Name:         types.StringValue("domain-updated"),
		Description:  types.StringValue("updated"),
		Enabled:      types.BoolValue(false),
		DataPlaneID:  types.StringValue("default"),
		SettingsJSON: types.StringValue(`{"tags":["team-b"],"saml":{"enabled":false}}`),
		OIDC: &OIDCModel{
			AllowLocalhostRedirectURI:        types.BoolValue(false),
			AllowHTTPSchemeRedirectURI:       types.BoolValue(true),
			AllowWildcardRedirectURI:         types.BoolValue(false),
			DynamicClientRegistrationEnabled: types.BoolValue(false),
		},
		LoginSettings: &LoginSettingsModel{
			RegisterEnabled:        types.BoolValue(false),
			ForgotPasswordEnabled:  types.BoolValue(true),
			IdentifierFirstEnabled: types.BoolValue(false),
		},
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: readResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if !reflect.DeepEqual(methods, []string{"create", "update", "read", "read", "update", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if _, ok := bodies[0]["enabled"]; ok {
		t.Fatalf("create body should be minimal: %#v", bodies[0])
	}
	postCreate := bodies[1]
	if postCreate["enabled"] != true {
		t.Fatalf("post-create enabled = %#v", postCreate["enabled"])
	}
	if _, ok := postCreate["id"]; ok {
		t.Fatalf("post-create body should not include id: %#v", postCreate)
	}
	updateBody := bodies[2]
	for _, field := range []string{"requiredPermissions", "uma"} {
		if _, ok := updateBody[field]; !ok {
			t.Fatalf("update body missing preserved field %q: %#v", field, updateBody)
		}
	}
	if tags, ok := updateBody["tags"].([]interface{}); !ok || len(tags) != 1 || tags[0] != "team-b" {
		t.Fatalf("update tags = %#v", updateBody["tags"])
	}
	if saml, ok := updateBody["saml"].(map[string]interface{}); !ok || saml["enabled"] != false {
		t.Fatalf("update saml = %#v", updateBody["saml"])
	}
	oidc := updateBody["oidc"].(map[string]interface{})
	crs := oidc["clientRegistrationSettings"].(map[string]interface{})
	if crs["allowHttpSchemeRedirectUri"] != true || crs["preservedNestedFlag"] != true {
		t.Fatalf("update oidc client registration settings = %#v", crs)
	}
	loginSettings := updateBody["loginSettings"].(map[string]interface{})
	if loginSettings["forgotPasswordEnabled"] != true || loginSettings["preservedLoginSetting"] != true {
		t.Fatalf("update login settings = %#v", loginSettings)
	}
}

func domainResponse(id string, body map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"id": id,
	}
	for key, value := range body {
		result[key] = value
	}
	return result
}

func domainPlan(t *testing.T, schema resourceschema.Schema, model DomainModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
