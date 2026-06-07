package applicationemail

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
	NewApplicationEmailResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_application_email"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewApplicationEmailResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "application_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "template", true, false, false)
	assertBoolAttribute(t, resp.Schema.Attributes, "enabled", false, true, true)
	assertStringAttribute(t, resp.Schema.Attributes, "from", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "from_name", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "subject", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "content", true, false, false)
	assertInt64Attribute(t, resp.Schema.Attributes, "expires_after", true, false, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationEmailResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestBuildBodyIncludesConfiguredFromName(t *testing.T) {
	model := ApplicationEmailModel{
		Enabled:      types.BoolValue(true),
		From:         types.StringValue("noreply@example.test"),
		FromName:     types.StringValue("Support"),
		Subject:      types.StringValue("Confirm"),
		Content:      types.StringValue("<html>Confirm</html>"),
		ExpiresAfter: types.Int64Value(86400),
	}

	body := (&ApplicationEmailResource{}).buildBody(model, nil)

	if body["fromName"] != "Support" {
		t.Fatalf("fromName = %#v, want Support", body["fromName"])
	}
	if body["enabled"] != true || body["from"] != "noreply@example.test" || body["subject"] != "Confirm" || body["content"] != "<html>Confirm</html>" || body["expiresAfter"] != int64(86400) {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestBuildBodyClearsRemovedFromName(t *testing.T) {
	plan := ApplicationEmailModel{
		Enabled:      types.BoolValue(true),
		From:         types.StringValue("noreply@example.test"),
		Subject:      types.StringValue("Confirm"),
		Content:      types.StringValue("<html>Confirm</html>"),
		ExpiresAfter: types.Int64Value(86400),
	}
	state := ApplicationEmailModel{
		FromName: types.StringValue("Support"),
	}

	body := (&ApplicationEmailResource{}).buildBody(plan, &state)

	if body["fromName"] != "" {
		t.Fatalf("fromName = %#v, want empty string", body["fromName"])
	}
}

func TestReadIntoModelMapsApplicationEmailResponse(t *testing.T) {
	model := ApplicationEmailModel{}

	(&ApplicationEmailResource{}).readIntoModel(&model, map[string]interface{}{
		"id":           "email-id",
		"template":     "registration_confirmation",
		"enabled":      false,
		"from":         "noreply@example.test",
		"fromName":     "Support",
		"subject":      "Confirm",
		"content":      "<html>Confirm</html>",
		"expiresAfter": float64(3600),
	})

	if model.ID.ValueString() != "email-id" {
		t.Fatalf("id = %q, want email-id", model.ID.ValueString())
	}
	if model.Template.ValueString() != "REGISTRATION_CONFIRMATION" {
		t.Fatalf("template = %q, want REGISTRATION_CONFIRMATION", model.Template.ValueString())
	}
	if model.Enabled.ValueBool() {
		t.Fatal("enabled = true, want false")
	}
	if model.From.ValueString() != "noreply@example.test" {
		t.Fatalf("from = %q, want noreply@example.test", model.From.ValueString())
	}
	if model.FromName.ValueString() != "Support" {
		t.Fatalf("from_name = %q, want Support", model.FromName.ValueString())
	}
	if model.Subject.ValueString() != "Confirm" {
		t.Fatalf("subject = %q, want Confirm", model.Subject.ValueString())
	}
	if model.Content.ValueString() != "<html>Confirm</html>" {
		t.Fatalf("content = %q, want HTML", model.Content.ValueString())
	}
	if model.ExpiresAfter.ValueInt64() != 3600 {
		t.Fatalf("expires_after = %d, want 3600", model.ExpiresAfter.ValueInt64())
	}
}

func TestReadIntoModelKeepsExistingFromNameWhenAPIOmitsEmptyValue(t *testing.T) {
	model := ApplicationEmailModel{
		FromName: types.StringValue("Support"),
	}

	(&ApplicationEmailResource{}).readIntoModel(&model, map[string]interface{}{
		"fromName": "",
	})

	if model.FromName.ValueString() != "Support" {
		t.Fatalf("from_name = %q, want preserved Support", model.FromName.ValueString())
	}
}

func TestApplicationEmailCRUDClearsRemovedFromName(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			methods = append(methods, "create")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(applicationEmailResponse("email-123", body))
		case http.MethodGet:
			if r.URL.Query().Get("template") != "RESET_PASSWORD" {
				t.Fatalf("template query = %q", r.URL.RawQuery)
			}
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           "email-123",
				"template":     "reset_password",
				"enabled":      true,
				"from":         "noreply@example.test",
				"fromName":     "Support",
				"subject":      "Reset",
				"content":      "<html>Reset</html>",
				"expiresAfter": float64(3600),
			})
		default:
			t.Fatalf("collection method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails/email-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(applicationEmailResponse("email-123", body))
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationEmailResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := applicationEmailPlan(t, schemaResp.Schema, ApplicationEmailModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("RESET_PASSWORD"),
		Enabled:       types.BoolValue(true),
		From:          types.StringValue("noreply@example.test"),
		FromName:      types.StringValue("Support"),
		Subject:       types.StringValue("Reset"),
		Content:       types.StringValue("<html>Reset</html>"),
		ExpiresAfter:  types.Int64Value(3600),
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

	updatePlan := applicationEmailPlan(t, schemaResp.Schema, ApplicationEmailModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("RESET_PASSWORD"),
		Enabled:       types.BoolValue(false),
		From:          types.StringValue("noreply@example.test"),
		FromName:      types.StringNull(),
		Subject:       types.StringValue("Reset updated"),
		Content:       types.StringValue("<html>Updated</html>"),
		ExpiresAfter:  types.Int64Value(7200),
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

	if !reflect.DeepEqual(methods, []string{"create", "read", "update", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if got := bodies[0]["template"]; got != "RESET_PASSWORD" {
		t.Fatalf("create template = %#v, want RESET_PASSWORD", got)
	}
	if got := bodies[1]["fromName"]; got != "" {
		t.Fatalf("update fromName = %#v, want clear string", got)
	}
	if _, ok := bodies[1]["template"]; ok {
		t.Fatalf("update body should not include immutable template: %#v", bodies[1])
	}
}

func TestApplicationEmailReadRemovesMissingTemplateAndDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("collection method = %s, want GET", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails/email-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationEmailResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ApplicationEmailModel{
		ID:            types.StringValue("email-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("RESET_PASSWORD"),
		Enabled:       types.BoolValue(true),
		From:          types.StringValue("noreply@example.test"),
		FromName:      types.StringValue("Support"),
		Subject:       types.StringValue("Reset"),
		Content:       types.StringValue("<html>Reset</html>"),
		ExpiresAfter:  types.Int64Value(3600),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing application email to remove state, got %#v", readResp.State.Raw)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestApplicationEmailUpdateAndDeleteReportRemoteErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails/email-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut, http.MethodDelete:
			http.Error(w, "remote error", http.StatusInternalServerError)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationEmailResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ApplicationEmailModel{
		ID:            types.StringValue("email-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("RESET_PASSWORD"),
		Enabled:       types.BoolValue(true),
		From:          types.StringValue("noreply@example.test"),
		FromName:      types.StringValue("Support"),
		Subject:       types.StringValue("Reset"),
		Content:       types.StringValue("<html>Reset</html>"),
		ExpiresAfter:  types.Int64Value(3600),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	plan := applicationEmailPlan(t, schemaResp.Schema, ApplicationEmailModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("RESET_PASSWORD"),
		Enabled:       types.BoolValue(false),
		From:          types.StringValue("noreply@example.test"),
		Subject:       types.StringValue("Reset updated"),
		Content:       types.StringValue("<html>Updated</html>"),
		ExpiresAfter:  types.Int64Value(7200),
	})

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func applicationEmailResponse(id string, body map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"id":           id,
		"template":     "RESET_PASSWORD",
		"enabled":      body["enabled"],
		"from":         body["from"],
		"subject":      body["subject"],
		"content":      body["content"],
		"expiresAfter": body["expiresAfter"],
	}
	if fromName, ok := body["fromName"]; ok {
		result["fromName"] = fromName
	}
	return result
}

func applicationEmailPlan(t *testing.T, schema resourceschema.Schema, model ApplicationEmailModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}

func assertBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.BoolAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}

func assertInt64Attribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.Int64Attribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.Int64Attribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
