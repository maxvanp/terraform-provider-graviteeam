package emailtemplate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewEmailTemplateResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_email_template"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewEmailTemplateResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "template", "from", "subject", "content", "expires_after"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"application_id", "enabled", "from_name"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "enabled"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
}

func TestEmailTemplateValidatorDescriptions(t *testing.T) {
	t.Parallel()

	validator := emailTemplateValidator{}
	description := validator.Description(context.Background())
	if description == "" {
		t.Fatal("expected description")
	}
	if got := validator.MarkdownDescription(context.Background()); got != description {
		t.Fatalf("markdown description = %q, want %q", got, description)
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&EmailTemplateResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&EmailTemplateResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	resource := &EmailTemplateResource{}
	plan := EmailTemplateModel{
		Enabled:      types.BoolValue(false),
		From:         types.StringValue("noreply@example.com"),
		FromName:     types.StringValue("Example"),
		Subject:      types.StringValue("Welcome"),
		Content:      types.StringValue("<html>Hello</html>"),
		ExpiresAfter: types.Int64Value(3600),
	}

	got := resource.buildBody(plan, nil)
	want := map[string]interface{}{
		"enabled":      false,
		"from":         "noreply@example.com",
		"fromName":     "Example",
		"subject":      "Welcome",
		"content":      "<html>Hello</html>",
		"expiresAfter": int64(3600),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyClearsRemovedFromName(t *testing.T) {
	t.Parallel()

	resource := &EmailTemplateResource{}
	plan := EmailTemplateModel{
		Enabled:      types.BoolValue(true),
		From:         types.StringValue("noreply@example.com"),
		FromName:     types.StringNull(),
		Subject:      types.StringValue("Reset"),
		Content:      types.StringValue("<html>Reset</html>"),
		ExpiresAfter: types.Int64Value(7200),
	}
	state := EmailTemplateModel{
		FromName: types.StringValue("Old Sender"),
	}

	got := resource.buildBody(plan, &state)
	if got["fromName"] != "" {
		t.Fatalf("fromName = %#v, want empty string clear marker", got["fromName"])
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	resource := &EmailTemplateResource{}
	model := EmailTemplateModel{
		FromName: types.StringValue("Existing Sender"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":           "email-1",
		"template":     "reset_password",
		"enabled":      false,
		"from":         "noreply@example.com",
		"fromName":     "API Sender",
		"subject":      "Reset",
		"content":      "<html>Reset</html>",
		"expiresAfter": float64(86400),
	})

	assertString(t, model.ID, "id", "email-1")
	assertString(t, model.Template, "template", "RESET_PASSWORD")
	assertBool(t, model.Enabled, "enabled", false)
	assertString(t, model.From, "from", "noreply@example.com")
	assertString(t, model.FromName, "fromName", "API Sender")
	assertString(t, model.Subject, "subject", "Reset")
	assertString(t, model.Content, "content", "<html>Reset</html>")
	assertInt64(t, model.ExpiresAfter, "expiresAfter", 86400)
}

func TestReadIntoModelPreservesExistingFromNameWhenAPIValueEmpty(t *testing.T) {
	t.Parallel()

	resource := &EmailTemplateResource{}
	model := EmailTemplateModel{
		FromName: types.StringValue("Existing Sender"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"fromName": "",
	})

	assertString(t, model.FromName, "fromName", "Existing Sender")
}

func TestEmailTemplateValidator(t *testing.T) {
	t.Parallel()

	v := emailTemplateValidator{}

	for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
		resp := validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: value,
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("null/unknown template diagnostics: %v", resp.Diagnostics)
		}
	}

	validResp := validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringValue("RESET_PASSWORD"),
	}, &validResp)
	if validResp.Diagnostics.HasError() {
		t.Fatalf("valid template diagnostics: %v", validResp.Diagnostics)
	}

	invalidResp := validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringValue("NOT_A_TEMPLATE"),
	}, &invalidResp)
	if !invalidResp.Diagnostics.HasError() {
		t.Fatalf("invalid template should produce diagnostics")
	}
}

func TestEmailTemplateCRUDClearsRemovedFromName(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/emails", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			methods = append(methods, "create")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(emailResponse("email-123", body))
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/emails/email-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(emailResponse("email-123", body))
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &EmailTemplateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := emailTemplatePlan(t, schemaResp.Schema, EmailTemplateModel{
		DomainID:     types.StringValue("domain-123"),
		Template:     types.StringValue("RESET_PASSWORD"),
		Enabled:      types.BoolValue(true),
		From:         types.StringValue("noreply@example.test"),
		FromName:     types.StringValue("Support"),
		Subject:      types.StringValue("Reset"),
		Content:      types.StringValue("<html>Reset</html>"),
		ExpiresAfter: types.Int64Value(3600),
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

	updatePlan := emailTemplatePlan(t, schemaResp.Schema, EmailTemplateModel{
		DomainID:     types.StringValue("domain-123"),
		Template:     types.StringValue("RESET_PASSWORD"),
		Enabled:      types.BoolValue(false),
		From:         types.StringValue("noreply@example.test"),
		FromName:     types.StringNull(),
		Subject:      types.StringValue("Reset updated"),
		Content:      types.StringValue("<html>Updated</html>"),
		ExpiresAfter: types.Int64Value(7200),
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

func TestEmailTemplateUsesApplicationScopedEndpoints(t *testing.T) {
	var paths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.Method {
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(emailResponse("email-123", body))
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           "email-123",
				"template":     "reset_password",
				"enabled":      true,
				"from":         "noreply@example.test",
				"subject":      "Reset",
				"content":      "<html>Reset</html>",
				"expiresAfter": float64(3600),
			})
		default:
			t.Fatalf("collection method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails/email-123", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.Method {
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(emailResponse("email-123", body))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &EmailTemplateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := emailTemplatePlan(t, schemaResp.Schema, EmailTemplateModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("RESET_PASSWORD"),
		Enabled:       types.BoolValue(true),
		From:          types.StringValue("noreply@example.test"),
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
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: createPlan, State: readResp.State}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	wantPaths := []string{
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails/email-123",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/emails/email-123",
	}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
}

func TestEmailTemplateReadRemovesMissingTemplateAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{
			name:       "missing template",
			statusCode: http.StatusNotFound,
			wantRemove: true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantRemove: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/emails", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &EmailTemplateResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := emailTemplateState(t, schemaResp.Schema, EmailTemplateModel{
				ID:           types.StringValue("email-123"),
				DomainID:     types.StringValue("domain-123"),
				Template:     types.StringValue("RESET_PASSWORD"),
				Enabled:      types.BoolValue(true),
				From:         types.StringValue("noreply@example.test"),
				Subject:      types.StringValue("Reset"),
				Content:      types.StringValue("<html>Reset</html>"),
				ExpiresAfter: types.Int64Value(3600),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing email template to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestEmailTemplateReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/emails", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/emails/email-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			http.Error(w, "update failed", http.StatusInternalServerError)
		case http.MethodDelete:
			http.Error(w, "delete failed", http.StatusInternalServerError)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &EmailTemplateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := emailTemplatePlan(t, schemaResp.Schema, EmailTemplateModel{
		DomainID:     types.StringValue("domain-123"),
		Template:     types.StringValue("RESET_PASSWORD"),
		Enabled:      types.BoolValue(true),
		From:         types.StringValue("noreply@example.test"),
		Subject:      types.StringValue("Reset"),
		Content:      types.StringValue("<html>Reset</html>"),
		ExpiresAfter: types.Int64Value(3600),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := emailTemplateState(t, schemaResp.Schema, EmailTemplateModel{
		ID:           types.StringValue("email-123"),
		DomainID:     types.StringValue("domain-123"),
		Template:     types.StringValue("RESET_PASSWORD"),
		Enabled:      types.BoolValue(true),
		From:         types.StringValue("noreply@example.test"),
		Subject:      types.StringValue("Reset"),
		Content:      types.StringValue("<html>Reset</html>"),
		ExpiresAfter: types.Int64Value(3600),
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

func TestEmailTemplateDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/emails/email-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &EmailTemplateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := emailTemplateState(t, schemaResp.Schema, baseEmailTemplateModel())

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestEmailTemplateUpdateReportsInvalidPlanAndStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &EmailTemplateResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"domain_id":      tftypes.Number,
			"application_id": tftypes.String,
			"template":       tftypes.String,
			"enabled":        tftypes.Bool,
			"from":           tftypes.String,
			"from_name":      tftypes.String,
			"subject":        tftypes.String,
			"content":        tftypes.String,
			"expires_after":  tftypes.Number,
		}},
		map[string]tftypes.Value{
			"id":             tftypes.NewValue(tftypes.String, "email-123"),
			"domain_id":      tftypes.NewValue(tftypes.Number, 123),
			"application_id": tftypes.NewValue(tftypes.String, nil),
			"template":       tftypes.NewValue(tftypes.String, "RESET_PASSWORD"),
			"enabled":        tftypes.NewValue(tftypes.Bool, true),
			"from":           tftypes.NewValue(tftypes.String, "noreply@example.test"),
			"from_name":      tftypes.NewValue(tftypes.String, nil),
			"subject":        tftypes.NewValue(tftypes.String, "Reset"),
			"content":        tftypes.NewValue(tftypes.String, "<html>Reset</html>"),
			"expires_after":  tftypes.NewValue(tftypes.Number, int64(3600)),
		},
	)

	invalidPlanResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		State: emailTemplateState(t, schemaResp.Schema, baseEmailTemplateModel()),
	}, invalidPlanResp)
	if !invalidPlanResp.Diagnostics.HasError() {
		t.Fatal("expected invalid plan diagnostics")
	}

	invalidStateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  emailTemplatePlan(t, schemaResp.Schema, baseEmailTemplateModel()),
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, invalidStateResp)
	if !invalidStateResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestEmailTemplateImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&EmailTemplateResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain/app/template/extra",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestEmailTemplateImportStateSetsDomainScopedAttributes(t *testing.T) {
	resp := emailTemplateImportResponse(t)

	(&EmailTemplateResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/RESET_PASSWORD",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state EmailTemplateModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get import state: %#v", diags)
	}
	assertString(t, state.DomainID, "domain_id", "domain-123")
	assertString(t, state.Template, "template", "RESET_PASSWORD")
}

func TestEmailTemplateImportStateSetsApplicationScopedAttributes(t *testing.T) {
	resp := emailTemplateImportResponse(t)

	(&EmailTemplateResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/app-123/RESET_PASSWORD",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state EmailTemplateModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get import state: %#v", diags)
	}
	assertString(t, state.DomainID, "domain_id", "domain-123")
	assertString(t, state.ApplicationID, "application_id", "app-123")
	assertString(t, state.Template, "template", "RESET_PASSWORD")
}

func emailTemplateImportResponse(t *testing.T) resource.ImportStateResponse {
	t.Helper()

	var schemaResp resource.SchemaResponse
	(&EmailTemplateResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	return resource.ImportStateResponse{State: emailTemplateState(t, schemaResp.Schema, baseEmailTemplateModel())}
}

func baseEmailTemplateModel() EmailTemplateModel {
	return EmailTemplateModel{
		ID:            types.StringValue("email-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringNull(),
		Template:      types.StringValue("RESET_PASSWORD"),
		Enabled:       types.BoolValue(true),
		From:          types.StringValue("noreply@example.test"),
		FromName:      types.StringNull(),
		Subject:       types.StringValue("Reset"),
		Content:       types.StringValue("<html>Reset</html>"),
		ExpiresAfter:  types.Int64Value(3600),
	}
}

func emailResponse(id string, body map[string]interface{}) map[string]interface{} {
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

func emailTemplatePlan(t *testing.T, schema resourceschema.Schema, model EmailTemplateModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func emailTemplateState(t *testing.T, schema resourceschema.Schema, model EmailTemplateModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func assertString(t *testing.T, value types.String, name string, want string) {
	t.Helper()
	if got := value.ValueString(); got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}

func assertInt64(t *testing.T, value types.Int64, name string, want int64) {
	t.Helper()
	if got := value.ValueInt64(); got != want {
		t.Fatalf("%s = %d, want %d", name, got, want)
	}
}

func assertBool(t *testing.T, value types.Bool, name string, want bool) {
	t.Helper()
	if got := value.ValueBool(); got != want {
		t.Fatalf("%s = %t, want %t", name, got, want)
	}
}
