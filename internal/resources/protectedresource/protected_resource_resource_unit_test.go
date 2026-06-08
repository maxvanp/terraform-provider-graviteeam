package protectedresource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewProtectedResourceResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_protected_resource"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewProtectedResourceResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "type", "resource_identifiers"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"description", "settings_json"} {
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
	if attr := resp.Schema.Attributes["client_secret"]; attr == nil || !attr.IsSensitive() {
		t.Fatalf("client_secret should be sensitive")
	}
	if _, ok := resp.Schema.Blocks["feature"]; !ok {
		t.Fatalf("missing feature block")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ProtectedResourceResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ProtectedResourceResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestProtectedResourceBuildCreateBodyIncludesTypeAndSettings(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	plan := ProtectedResourceModel{
		Name:                types.StringValue("mcp"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.com/mcp")},
		SettingsJSON:        types.StringValue(`{"authorizationServer":"as-1","enabled":true}`),
		Features: []ProtectedResourceFeatureModel{
			{
				Key:         types.StringValue("list_items"),
				Type:        types.StringValue("MCP_TOOL"),
				Description: types.StringValue("List items"),
				Scopes:      []types.String{types.StringValue("openid"), types.StringValue("profile")},
			},
		},
	}

	body, err := resource.buildCreateBody(plan)
	if err != nil {
		t.Fatal(err)
	}

	if body["type"] != "MCP_SERVER" {
		t.Fatalf("type = %#v, want MCP_SERVER", body["type"])
	}

	settings, ok := body["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("settings = %#v, want object", body["settings"])
	}
	if settings["authorizationServer"] != "as-1" || settings["enabled"] != true {
		t.Fatalf("settings = %#v", settings)
	}

	features, ok := body["features"].([]map[string]interface{})
	if !ok || len(features) != 1 {
		t.Fatalf("features = %#v, want one feature", body["features"])
	}
	wantScopes := []string{"openid", "profile"}
	if !reflect.DeepEqual(features[0]["scopes"], wantScopes) {
		t.Fatalf("feature scopes = %#v, want %#v", features[0]["scopes"], wantScopes)
	}
}

func TestProtectedResourceBuildUpdateBodyClearsRemovedOptionalCollections(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	plan := ProtectedResourceModel{
		Name:                types.StringValue("updated"),
		Description:         types.StringValue("updated description"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.com/mcp")},
		SettingsJSON:        types.StringNull(),
		Features:            nil,
	}
	state := &ProtectedResourceModel{
		SettingsJSON: types.StringValue(`{"existing":true}`),
		Features: []ProtectedResourceFeatureModel{
			{
				Key:  types.StringValue("existing"),
				Type: types.StringValue("MCP_TOOL"),
			},
		},
	}

	body, err := resource.buildUpdateBody(plan, state)
	if err != nil {
		t.Fatal(err)
	}

	features, ok := body["features"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected removed feature block to be sent as an empty feature list, got %#v", body["features"])
	}
	if len(features) != 0 {
		t.Fatalf("expected empty feature list, got %#v", features)
	}

	settings, ok := body["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected removed settings_json to be sent as an empty settings object, got %#v", body["settings"])
	}
	if len(settings) != 0 {
		t.Fatalf("expected empty settings object, got %#v", settings)
	}
}

func TestProtectedResourceBuildUpdateBodyRejectsInvalidSettingsJSON(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	plan := ProtectedResourceModel{
		Name:                types.StringValue("mcp"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.com/mcp")},
		SettingsJSON:        types.StringValue(`{"broken":`),
	}

	_, err := resource.buildUpdateBody(plan, nil)
	if err == nil {
		t.Fatal("expected invalid settings_json error")
	}
	if !strings.Contains(err.Error(), "settings_json must be a valid JSON object") {
		t.Fatalf("error = %q", err)
	}
}

func TestProtectedResourceCreateReportsInvalidSettingsJSON(t *testing.T) {
	resourceUnderTest := &ProtectedResourceResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := protectedResourcePlan(t, schemaResp.Schema, ProtectedResourceModel{
		DomainID:            types.StringValue("domain-123"),
		Name:                types.StringValue("mcp"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.test/mcp")},
		SettingsJSON:        types.StringValue(`{"enabled":`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected invalid settings diagnostics")
	}
}

func TestProtectedResourceReadIntoModelMapsAPIFields(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	model := ProtectedResourceModel{}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":                  "resource-1",
		"name":                "mcp",
		"description":         "MCP server",
		"type":                "mcp_server",
		"clientId":            "client-1",
		"resourceIdentifiers": []interface{}{"https://api.example.com/mcp", 42, "https://api.example.com/secondary"},
		"features": []interface{}{
			map[string]interface{}{
				"key":         "list_items",
				"type":        "mcp_tool",
				"description": "List items",
				"scopes":      []interface{}{"openid", true, "profile"},
			},
			"ignored",
		},
	})

	if model.ID.ValueString() != "resource-1" ||
		model.Name.ValueString() != "mcp" ||
		model.Description.ValueString() != "MCP server" ||
		model.Type.ValueString() != "MCP_SERVER" ||
		model.ClientID.ValueString() != "client-1" {
		t.Fatalf("model = %#v", model)
	}

	wantIdentifiers := []types.String{types.StringValue("https://api.example.com/mcp"), types.StringValue("https://api.example.com/secondary")}
	if !reflect.DeepEqual(model.ResourceIdentifiers, wantIdentifiers) {
		t.Fatalf("resource identifiers = %#v, want %#v", model.ResourceIdentifiers, wantIdentifiers)
	}

	if len(model.Features) != 1 {
		t.Fatalf("features = %#v, want one valid feature", model.Features)
	}
	feature := model.Features[0]
	if feature.Key.ValueString() != "list_items" ||
		feature.Type.ValueString() != "MCP_TOOL" ||
		feature.Description.ValueString() != "List items" {
		t.Fatalf("feature = %#v", feature)
	}
	wantScopes := []types.String{types.StringValue("openid"), types.StringValue("profile")}
	if !reflect.DeepEqual(feature.Scopes, wantScopes) {
		t.Fatalf("feature scopes = %#v, want %#v", feature.Scopes, wantScopes)
	}
}

func TestProtectedResourceReadSecretIntoModelMapsCreationSecrets(t *testing.T) {
	t.Parallel()

	resource := &ProtectedResourceResource{}
	model := ProtectedResourceModel{}

	resource.readSecretIntoModel(&model, map[string]interface{}{
		"id":           "resource-1",
		"clientId":     "client-1",
		"clientSecret": "secret-1",
	})

	if model.ID.ValueString() != "resource-1" ||
		model.ClientID.ValueString() != "client-1" ||
		model.ClientSecret.ValueString() != "secret-1" {
		t.Fatalf("model = %#v", model)
	}
}

func TestProtectedResourceCRUDPreservesSecretAndClearsRemovedSettings(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "resource-123",
			"clientId":     "client-123",
			"clientSecret": "clear-secret",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if got := r.URL.Query().Get("type"); got != "MCP_SERVER" {
				t.Fatalf("type query = %q, want MCP_SERVER", got)
			}
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(protectedResourceResponse("resource-123", map[string]interface{}{
				"name":                "mcp",
				"description":         "MCP server",
				"type":                "MCP_SERVER",
				"clientId":            "client-123",
				"resourceIdentifiers": []interface{}{"https://api.example.test/mcp"},
				"features": []interface{}{
					map[string]interface{}{
						"key":         "list_items",
						"type":        "MCP_TOOL",
						"description": "List items",
						"scopes":      []interface{}{"openid"},
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
			_ = json.NewEncoder(w).Encode(protectedResourceResponse("resource-123", map[string]interface{}{
				"name":                body["name"],
				"type":                "MCP_SERVER",
				"clientId":            "client-123",
				"resourceIdentifiers": body["resourceIdentifiers"],
				"features":            body["features"],
			}))
		case http.MethodDelete:
			if got := r.URL.Query().Get("type"); got != "MCP_SERVER" {
				t.Fatalf("type query = %q, want MCP_SERVER", got)
			}
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ProtectedResourceResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := protectedResourcePlan(t, schemaResp.Schema, ProtectedResourceModel{
		DomainID:            types.StringValue("domain-123"),
		Name:                types.StringValue("mcp"),
		Description:         types.StringValue("MCP server"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.test/mcp")},
		SettingsJSON:        types.StringValue(`{"authorizationServer":"as-1"}`),
		Features: []ProtectedResourceFeatureModel{
			{
				Key:         types.StringValue("list_items"),
				Type:        types.StringValue("MCP_TOOL"),
				Description: types.StringValue("List items"),
				Scopes:      []types.String{types.StringValue("openid")},
			},
		},
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createdState ProtectedResourceModel
	if diags := createResp.State.Get(context.Background(), &createdState); diags.HasError() {
		t.Fatalf("get created state: %#v", diags)
	}
	if createdState.ClientSecret.ValueString() != "clear-secret" {
		t.Fatalf("client_secret = %q, want clear-secret", createdState.ClientSecret.ValueString())
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState ProtectedResourceModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if readState.ClientSecret.ValueString() != "clear-secret" {
		t.Fatalf("read client_secret = %q, want preserved clear-secret", readState.ClientSecret.ValueString())
	}
	if readState.SettingsJSON.ValueString() != `{"authorizationServer":"as-1"}` {
		t.Fatalf("read settings_json = %q, want preserved plan settings", readState.SettingsJSON.ValueString())
	}

	updatePlan := protectedResourcePlan(t, schemaResp.Schema, ProtectedResourceModel{
		DomainID:            types.StringValue("domain-123"),
		Name:                types.StringValue("mcp-updated"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.test/mcp-updated")},
		SettingsJSON:        types.StringNull(),
		Features:            nil,
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: readResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	var updatedState ProtectedResourceModel
	if diags := updateResp.State.Get(context.Background(), &updatedState); diags.HasError() {
		t.Fatalf("get updated state: %#v", diags)
	}
	if !updatedState.SettingsJSON.IsNull() {
		t.Fatalf("updated settings_json = %#v, want null after explicit removal", updatedState.SettingsJSON)
	}
	if updatedState.ClientSecret.ValueString() != "clear-secret" {
		t.Fatalf("updated client_secret = %q, want preserved clear-secret", updatedState.ClientSecret.ValueString())
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if !reflect.DeepEqual(methods, []string{"create", "read", "update", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if got := bodies[0]["type"]; got != "MCP_SERVER" {
		t.Fatalf("create type = %#v, want MCP_SERVER", got)
	}
	createSettings, ok := bodies[0]["settings"].(map[string]interface{})
	if !ok || createSettings["authorizationServer"] != "as-1" {
		t.Fatalf("create settings = %#v", bodies[0]["settings"])
	}
	if _, ok := bodies[1]["type"]; ok {
		t.Fatalf("update body should not send immutable type: %#v", bodies[1])
	}
	updateSettings, ok := bodies[1]["settings"].(map[string]interface{})
	if !ok || len(updateSettings) != 0 {
		t.Fatalf("update settings = %#v, want empty object", bodies[1]["settings"])
	}
	if got, ok := bodies[1]["features"].([]interface{}); !ok || len(got) != 0 {
		t.Fatalf("update features = %#v, want empty list", bodies[1]["features"])
	}
}

func TestProtectedResourceReadRemovesMissingResource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ProtectedResourceResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := protectedResourceState(t, schemaResp.Schema, ProtectedResourceModel{
		ID:                  types.StringValue("resource-123"),
		DomainID:            types.StringValue("domain-123"),
		Name:                types.StringValue("mcp"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.test/mcp")},
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing protected resource to remove state, got %#v", readResp.State.Raw)
	}
}

func TestProtectedResourceCRUDReportsRemoteErrors(t *testing.T) {
	tests := map[string]struct {
		createStatus int
		itemStatus   int
		action       func(context.Context, *ProtectedResourceResource, tfsdk.Plan, tfsdk.State, resourceschema.Schema) bool
	}{
		"create": {
			createStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ProtectedResourceResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"read": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ProtectedResourceResource, _ tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
				r.Read(ctx, resource.ReadRequest{State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"update": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ProtectedResourceResource, plan tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"delete": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *ProtectedResourceResource, _ tfsdk.Plan, state tfsdk.State, _ resourceschema.Schema) bool {
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("collection method = %s, want POST", r.Method)
				}
				if tc.createStatus != 0 {
					http.Error(w, "remote error", tc.createStatus)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":           "resource-123",
					"clientId":     "client-123",
					"clientSecret": "clear-secret",
				})
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123", func(w http.ResponseWriter, r *http.Request) {
				if tc.itemStatus != 0 {
					http.Error(w, "remote error", tc.itemStatus)
					return
				}
				switch r.Method {
				case http.MethodGet, http.MethodPut:
					_ = json.NewEncoder(w).Encode(protectedResourceResponse("resource-123", map[string]interface{}{
						"name":                "mcp",
						"type":                "MCP_SERVER",
						"clientId":            "client-123",
						"resourceIdentifiers": []interface{}{"https://api.example.test/mcp"},
					}))
				case http.MethodDelete:
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Fatalf("item method = %s", r.Method)
				}
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &ProtectedResourceResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			model := ProtectedResourceModel{
				ID:                  types.StringValue("resource-123"),
				DomainID:            types.StringValue("domain-123"),
				Name:                types.StringValue("mcp"),
				Type:                types.StringValue("MCP_SERVER"),
				ResourceIdentifiers: []types.String{types.StringValue("https://api.example.test/mcp")},
				ClientSecret:        types.StringValue("clear-secret"),
			}
			plan := protectedResourcePlan(t, schemaResp.Schema, model)
			state := protectedResourceState(t, schemaResp.Schema, model)

			if !tc.action(context.Background(), resourceUnderTest, plan, state, schemaResp.Schema) {
				t.Fatal("expected diagnostics")
			}
		})
	}
}

func TestProtectedResourceCreateReadUpdateAndDeleteReportInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ProtectedResourceResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	stringSetType := tftypes.Set{ElementType: tftypes.String}
	featureType := tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"key":         tftypes.String,
		"type":        tftypes.String,
		"description": tftypes.String,
		"scopes":      stringSetType,
	}}}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":                   tftypes.String,
			"domain_id":            tftypes.Number,
			"name":                 tftypes.String,
			"description":          tftypes.String,
			"type":                 tftypes.String,
			"resource_identifiers": stringSetType,
			"settings_json":        tftypes.String,
			"client_id":            tftypes.String,
			"client_secret":        tftypes.String,
			"feature":              featureType,
		}},
		map[string]tftypes.Value{
			"id":                   tftypes.NewValue(tftypes.String, "resource-123"),
			"domain_id":            tftypes.NewValue(tftypes.Number, 123),
			"name":                 tftypes.NewValue(tftypes.String, "mcp"),
			"description":          tftypes.NewValue(tftypes.String, nil),
			"type":                 tftypes.NewValue(tftypes.String, "MCP_SERVER"),
			"resource_identifiers": tftypes.NewValue(stringSetType, []tftypes.Value{tftypes.NewValue(tftypes.String, "https://api.example.test/mcp")}),
			"settings_json":        tftypes.NewValue(tftypes.String, nil),
			"client_id":            tftypes.NewValue(tftypes.String, "client-123"),
			"client_secret":        tftypes.NewValue(tftypes.String, "secret-123"),
			"feature":              tftypes.NewValue(featureType, nil),
		},
	)

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
	}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestProtectedResourceUpdateReportsInvalidStateAfterValidPlan(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ProtectedResourceResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	stringSetType := tftypes.Set{ElementType: tftypes.String}
	featureType := tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"key":         tftypes.String,
		"type":        tftypes.String,
		"description": tftypes.String,
		"scopes":      stringSetType,
	}}}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":                   tftypes.String,
			"domain_id":            tftypes.Number,
			"name":                 tftypes.String,
			"description":          tftypes.String,
			"type":                 tftypes.String,
			"resource_identifiers": stringSetType,
			"settings_json":        tftypes.String,
			"client_id":            tftypes.String,
			"client_secret":        tftypes.String,
			"feature":              featureType,
		}},
		map[string]tftypes.Value{
			"id":                   tftypes.NewValue(tftypes.String, "resource-123"),
			"domain_id":            tftypes.NewValue(tftypes.Number, 123),
			"name":                 tftypes.NewValue(tftypes.String, "mcp"),
			"description":          tftypes.NewValue(tftypes.String, nil),
			"type":                 tftypes.NewValue(tftypes.String, "MCP_SERVER"),
			"resource_identifiers": tftypes.NewValue(stringSetType, []tftypes.Value{tftypes.NewValue(tftypes.String, "https://api.example.test/mcp")}),
			"settings_json":        tftypes.NewValue(tftypes.String, nil),
			"client_id":            tftypes.NewValue(tftypes.String, "client-123"),
			"client_secret":        tftypes.NewValue(tftypes.String, "secret-123"),
			"feature":              tftypes.NewValue(featureType, nil),
		},
	)
	plan := protectedResourcePlan(t, schemaResp.Schema, ProtectedResourceModel{
		ID:                  types.StringValue("resource-123"),
		DomainID:            types.StringValue("domain-123"),
		Name:                types.StringValue("mcp"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.test/mcp")},
		ClientID:            types.StringValue("client-123"),
		ClientSecret:        types.StringValue("secret-123"),
	})

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  plan,
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestProtectedResourceDeleteIgnoresMissingResource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ProtectedResourceResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := protectedResourceState(t, schemaResp.Schema, ProtectedResourceModel{
		ID:                  types.StringValue("resource-123"),
		DomainID:            types.StringValue("domain-123"),
		Name:                types.StringValue("mcp"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.test/mcp")},
	})

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestProtectedResourceImportStateRejectsInvalidID(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewProtectedResourceResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ProtectedResourceModel{
		ID:                  types.StringValue("placeholder"),
		DomainID:            types.StringValue("placeholder-domain"),
		Name:                types.StringValue("placeholder"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.test/mcp")},
		ClientID:            types.StringValue("placeholder-client"),
		ClientSecret:        types.StringValue("placeholder-secret"),
	}); diags.HasError() {
		t.Fatalf("set empty state: %#v", diags)
	}
	importResp := &resource.ImportStateResponse{State: state}

	NewProtectedResourceResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "resource-only",
	}, importResp)

	if !importResp.Diagnostics.HasError() {
		t.Fatal("expected invalid import diagnostics")
	}
}

func TestProtectedResourceImportStateSetsDomainAndID(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewProtectedResourceResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	importResp := &resource.ImportStateResponse{State: protectedResourceState(t, schemaResp.Schema, ProtectedResourceModel{
		ID:                  types.StringValue("placeholder"),
		DomainID:            types.StringValue("placeholder"),
		Name:                types.StringValue("mcp"),
		Type:                types.StringValue("MCP_SERVER"),
		ResourceIdentifiers: []types.String{types.StringValue("https://api.example.test/mcp")},
		ClientID:            types.StringValue("client-id"),
		ClientSecret:        types.StringValue("client-secret"),
	})}

	NewProtectedResourceResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/resource-123",
	}, importResp)

	if importResp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", importResp.Diagnostics)
	}
	var imported ProtectedResourceModel
	if diags := importResp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.ID.ValueString(), "resource-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func protectedResourceResponse(id string, body map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"id": id,
	}
	for key, value := range body {
		result[key] = value
	}
	return result
}

func protectedResourceState(t *testing.T, schema resourceschema.Schema, model ProtectedResourceModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func protectedResourcePlan(t *testing.T, schema resourceschema.Schema, model ProtectedResourceModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
