package protectedresourcesecret

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewProtectedResourceSecretResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_protected_resource_secret"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewProtectedResourceSecretResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "protected_resource_id", true, false, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "renew_trigger", false, true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "secret", false, false, true, true)
	assertStringAttribute(t, resp.Schema.Attributes, "settings_id", false, false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "expires_at", false, false, true, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ProtectedResourceSecretResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		id                      string
		wantDomainID            string
		wantProtectedResourceID string
		wantSecretID            string
		wantOK                  bool
	}{
		{
			name:                    "valid",
			id:                      "domain-1/protected-resource-1/secret-1",
			wantDomainID:            "domain-1",
			wantProtectedResourceID: "protected-resource-1",
			wantSecretID:            "secret-1",
			wantOK:                  true,
		},
		{
			name:   "missing part",
			id:     "domain-1/protected-resource-1",
			wantOK: false,
		},
		{
			name:   "extra part",
			id:     "domain-1/protected-resource-1/secret-1/extra",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotProtectedResourceID, gotSecretID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotProtectedResourceID != tt.wantProtectedResourceID {
				t.Fatalf("protected resource ID = %q, want %q", gotProtectedResourceID, tt.wantProtectedResourceID)
			}
			if gotSecretID != tt.wantSecretID {
				t.Fatalf("secret ID = %q, want %q", gotSecretID, tt.wantSecretID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := ProtectedResourceSecretModel{
		Name: types.StringValue("client-secret"),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name": "client-secret",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelUsesReturnedSecret(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":         "secret-1",
		"name":       "client-secret",
		"secret":     "clear-secret",
		"settingsId": "settings-1",
		"expiresAt":  "2026-06-07T12:00:00Z",
	}, types.StringValue("old-secret"))

	if got, want := model.ID.ValueString(), "secret-1"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.Name.ValueString(), "client-secret"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Secret.ValueString(), "clear-secret"; got != want {
		t.Fatalf("secret = %q, want %q", got, want)
	}
	if got, want := model.SettingsID.ValueString(), "settings-1"; got != want {
		t.Fatalf("settings ID = %q, want %q", got, want)
	}
	if got, want := model.ExpiresAt.ValueString(), "2026-06-07T12:00:00Z"; got != want {
		t.Fatalf("expires at = %q, want %q", got, want)
	}
}

func TestReadIntoModelPreservesExistingSecretWhenAPIOmitsClearValue(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":     "secret-1",
		"name":   "client-secret",
		"secret": "",
	}, types.StringValue("preserved-secret"))

	if got, want := model.Secret.ValueString(), "preserved-secret"; got != want {
		t.Fatalf("secret = %q, want preserved %q", got, want)
	}
	if !model.SettingsID.IsNull() {
		t.Fatalf("settings ID should be null, got %q", model.SettingsID.ValueString())
	}
	if !model.ExpiresAt.IsNull() {
		t.Fatalf("expires at should be null, got %q", model.ExpiresAt.ValueString())
	}
}

func TestReadIntoModelSetsNullSecretWhenNoPreservedValue(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":   "secret-1",
		"name": "client-secret",
	}, types.StringUnknown())

	if !model.Secret.IsNull() {
		t.Fatalf("secret should be null, got %q", model.Secret.ValueString())
	}
}

func TestShouldRenew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		planTrigger  types.String
		stateTrigger types.String
		want         bool
	}{
		{
			name:         "null plan does not renew",
			planTrigger:  types.StringNull(),
			stateTrigger: types.StringValue("old"),
			want:         false,
		},
		{
			name:         "unknown plan does not renew",
			planTrigger:  types.StringUnknown(),
			stateTrigger: types.StringValue("old"),
			want:         false,
		},
		{
			name:         "first configured trigger renews",
			planTrigger:  types.StringValue("new"),
			stateTrigger: types.StringNull(),
			want:         true,
		},
		{
			name:         "changed trigger renews",
			planTrigger:  types.StringValue("new"),
			stateTrigger: types.StringValue("old"),
			want:         true,
		},
		{
			name:         "same trigger does not renew",
			planTrigger:  types.StringValue("same"),
			stateTrigger: types.StringValue("same"),
			want:         false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldRenew(tt.planTrigger, tt.stateTrigger); got != tt.want {
				t.Fatalf("shouldRenew = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestProtectedResourceSecretCRUDPreservesAndRenewsSecret(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/secrets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			methods = append(methods, "create")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         "secret-123",
				"name":       body["name"],
				"secret":     "clear-secret",
				"settingsId": "settings-1",
				"expiresAt":  "2026-06-07T12:00:00Z",
			})
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":     "unrelated",
					"name":   "other-secret",
					"secret": "",
				},
				{
					"id":         "secret-123",
					"name":       "client-secret",
					"secret":     "",
					"settingsId": "settings-1",
					"expiresAt":  "2026-06-07T12:00:00Z",
				},
			})
		default:
			t.Fatalf("collection method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/secrets/secret-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s", r.Method)
		}
		methods = append(methods, "delete")
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/protected-resources/resource-123/secrets/secret-123/_renew", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("renew method = %s", r.Method)
		}
		methods = append(methods, "renew")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         "secret-123",
			"name":       "client-secret",
			"secret":     "renewed-secret",
			"settingsId": "settings-2",
			"expiresAt":  "2026-06-08T12:00:00Z",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ProtectedResourceSecretResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := protectedResourceSecretPlan(t, schemaResp.Schema, ProtectedResourceSecretModel{
		DomainID:            types.StringValue("domain-123"),
		ProtectedResourceID: types.StringValue("resource-123"),
		Name:                types.StringValue("client-secret"),
		RenewTrigger:        types.StringNull(),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState ProtectedResourceSecretModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if got, want := createState.Secret.ValueString(), "clear-secret"; got != want {
		t.Fatalf("created secret = %q, want %q", got, want)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState ProtectedResourceSecretModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := readState.Secret.ValueString(), "clear-secret"; got != want {
		t.Fatalf("read secret = %q, want preserved %q", got, want)
	}

	updatePlan := protectedResourceSecretPlan(t, schemaResp.Schema, ProtectedResourceSecretModel{
		DomainID:            types.StringValue("domain-123"),
		ProtectedResourceID: types.StringValue("resource-123"),
		Name:                types.StringValue("client-secret"),
		RenewTrigger:        types.StringValue("rotate-1"),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: readResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	var updateState ProtectedResourceSecretModel
	if diags := updateResp.State.Get(context.Background(), &updateState); diags.HasError() {
		t.Fatalf("get update state: %#v", diags)
	}
	if got, want := updateState.Secret.ValueString(), "renewed-secret"; got != want {
		t.Fatalf("renewed secret = %q, want %q", got, want)
	}
	if got, want := updateState.SettingsID.ValueString(), "settings-2"; got != want {
		t.Fatalf("settings ID = %q, want %q", got, want)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if !reflect.DeepEqual(methods, []string{"create", "read", "renew", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if !reflect.DeepEqual(bodies, []map[string]interface{}{{"name": "client-secret"}}) {
		t.Fatalf("create bodies = %#v", bodies)
	}
}

func protectedResourceSecretPlan(t *testing.T, schema resourceschema.Schema, model ProtectedResourceSecretModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed, sensitive bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed || attr.Sensitive != sensitive {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t sensitive:%t, want required:%t optional:%t computed:%t sensitive:%t",
			name, attr.Required, attr.Optional, attr.Computed, attr.Sensitive, required, optional, computed, sensitive)
	}
}
