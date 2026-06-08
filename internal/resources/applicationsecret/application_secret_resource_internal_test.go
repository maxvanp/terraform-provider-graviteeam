package applicationsecret

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
	NewApplicationSecretResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_application_secret"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewApplicationSecretResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "application_id", true, false, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "renew_trigger", false, true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "secret", false, false, true, true)
	assertStringAttribute(t, resp.Schema.Attributes, "settings_id", false, false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "expires_at", false, false, true, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationSecretResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ApplicationSecretResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if resourceUnderTest.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ApplicationSecretResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if resourceUnderTest.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestReadIntoModelUsesReturnedSecretAndMetadata(t *testing.T) {
	model := ApplicationSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":         "secret-id",
		"name":       "client secret",
		"secret":     "clear-secret",
		"settingsId": "settings-id",
		"expiresAt":  "2026-06-07T12:00:00Z",
	}, types.StringValue("old-secret"))

	if model.ID.ValueString() != "secret-id" {
		t.Fatalf("id = %q, want secret-id", model.ID.ValueString())
	}
	if model.Name.ValueString() != "client secret" {
		t.Fatalf("name = %q, want client secret", model.Name.ValueString())
	}
	if model.Secret.ValueString() != "clear-secret" {
		t.Fatalf("secret = %q, want clear-secret", model.Secret.ValueString())
	}
	if model.SettingsID.ValueString() != "settings-id" {
		t.Fatalf("settings_id = %q, want settings-id", model.SettingsID.ValueString())
	}
	if model.ExpiresAt.ValueString() != "2026-06-07T12:00:00Z" {
		t.Fatalf("expires_at = %q, want timestamp", model.ExpiresAt.ValueString())
	}
}

func TestReadIntoModelPreservesExistingSecretWhenAPIOmitsClearValue(t *testing.T) {
	model := ApplicationSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":     "secret-id",
		"name":   "client secret",
		"secret": "",
	}, types.StringValue("preserved-secret"))

	if model.Secret.ValueString() != "preserved-secret" {
		t.Fatalf("secret = %q, want preserved-secret", model.Secret.ValueString())
	}
	if !model.SettingsID.IsNull() {
		t.Fatalf("settings_id = %#v, want null", model.SettingsID)
	}
	if !model.ExpiresAt.IsNull() {
		t.Fatalf("expires_at = %#v, want null", model.ExpiresAt)
	}
}

func TestReadIntoModelNullsSecretWhenNoClearOrPreservedValue(t *testing.T) {
	model := ApplicationSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":   "secret-id",
		"name": "client secret",
	}, types.StringUnknown())

	if !model.Secret.IsNull() {
		t.Fatalf("secret = %#v, want null", model.Secret)
	}
}

func TestShouldRenew(t *testing.T) {
	tests := []struct {
		name  string
		plan  types.String
		state types.String
		want  bool
	}{
		{name: "plan null", plan: types.StringNull(), state: types.StringValue("old"), want: false},
		{name: "plan unknown", plan: types.StringUnknown(), state: types.StringValue("old"), want: false},
		{name: "state null", plan: types.StringValue("new"), state: types.StringNull(), want: true},
		{name: "state unknown", plan: types.StringValue("new"), state: types.StringUnknown(), want: true},
		{name: "same value", plan: types.StringValue("same"), state: types.StringValue("same"), want: false},
		{name: "changed value", plan: types.StringValue("new"), state: types.StringValue("old"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRenew(tt.plan, tt.state); got != tt.want {
				t.Fatalf("shouldRenew() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestApplicationSecretCRUDPreservesAndRenewsSecret(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	var bodies []map[string]interface{}
	var renewPaths []string
	var deletePaths []string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         "secret-123",
				"name":       body["name"],
				"secret":     "clear-secret",
				"settingsId": "settings-123",
				"expiresAt":  "2026-06-07T12:00:00Z",
			})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":         "secret-123",
					"name":       "client secret",
					"settingsId": "settings-123",
					"expiresAt":  "2026-06-07T12:00:00Z",
				},
			})
		default:
			t.Fatalf("unexpected application secret collection method %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets/secret-123/_renew", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected renew method %s", r.Method)
		}
		renewPaths = append(renewPaths, r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         "secret-123",
			"name":       "client secret",
			"secret":     "renewed-secret",
			"settingsId": "settings-123",
			"expiresAt":  "2026-06-08T12:00:00Z",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets/secret-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected application secret item method %s", r.Method)
		}
		deletePaths = append(deletePaths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationSecretResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := applicationSecretPlan(t, schemaResp.Schema, ApplicationSecretModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Name:          types.StringValue("client secret"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState ApplicationSecretModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := readState.Secret.ValueString(), "clear-secret"; got != want {
		t.Fatalf("read secret = %q, want preserved %q", got, want)
	}

	updatePlan := applicationSecretPlan(t, schemaResp.Schema, ApplicationSecretModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Name:          types.StringValue("client secret"),
		RenewTrigger:  types.StringValue("rotate-1"),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: readResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	var updateState ApplicationSecretModel
	if diags := updateResp.State.Get(context.Background(), &updateState); diags.HasError() {
		t.Fatalf("get update state: %#v", diags)
	}
	if got, want := updateState.Secret.ValueString(), "renewed-secret"; got != want {
		t.Fatalf("renewed secret = %q, want %q", got, want)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if want := []map[string]interface{}{{"name": "client secret"}}; !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies = %#v, want %#v", bodies, want)
	}
	if want := []string{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets/secret-123/_renew"}; !reflect.DeepEqual(renewPaths, want) {
		t.Fatalf("renew paths = %#v, want %#v", renewPaths, want)
	}
	if want := []string{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets/secret-123"}; !reflect.DeepEqual(deletePaths, want) {
		t.Fatalf("delete paths = %#v, want %#v", deletePaths, want)
	}
}

func TestApplicationSecretReadRemovesMissingSecretAndDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`[]`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets/secret-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationSecretResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := applicationSecretState(t, schemaResp.Schema, ApplicationSecretModel{
		ID:            types.StringValue("secret-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Name:          types.StringValue("client secret"),
		Secret:        types.StringValue("preserved-secret"),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing application secret to remove state, got %#v", readResp.State.Raw)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestApplicationSecretReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			http.Error(w, "create failed", http.StatusInternalServerError)
		case http.MethodGet:
			http.Error(w, "read failed", http.StatusInternalServerError)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets/secret-123/_renew", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "renew failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/secrets/secret-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationSecretResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := applicationSecretPlan(t, schemaResp.Schema, ApplicationSecretModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Name:          types.StringValue("client secret"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := applicationSecretState(t, schemaResp.Schema, ApplicationSecretModel{
		ID:            types.StringValue("secret-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Name:          types.StringValue("client secret"),
		Secret:        types.StringValue("preserved-secret"),
		RenewTrigger:  types.StringValue("old"),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	updatePlan := applicationSecretPlan(t, schemaResp.Schema, ApplicationSecretModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Name:          types.StringValue("client secret"),
		RenewTrigger:  types.StringValue("new"),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: state,
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestApplicationSecretImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&ApplicationSecretResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain/app",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestApplicationSecretImportStateSetsDomainApplicationAndID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewApplicationSecretResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: applicationSecretState(t, schemaResp.Schema, ApplicationSecretModel{
		ID:            types.StringValue("placeholder"),
		DomainID:      types.StringValue("placeholder"),
		ApplicationID: types.StringValue("placeholder"),
		Name:          types.StringValue("client secret"),
		Secret:        types.StringValue("preserved-secret"),
	})}

	(&ApplicationSecretResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/app-123/secret-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported ApplicationSecretModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.ApplicationID.ValueString(), "app-123"; got != want {
		t.Fatalf("application id = %q, want %q", got, want)
	}
	if got, want := imported.ID.ValueString(), "secret-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func applicationSecretPlan(t *testing.T, schema resourceschema.Schema, model ApplicationSecretModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func applicationSecretState(t *testing.T, schema resourceschema.Schema, model ApplicationSecretModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
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
