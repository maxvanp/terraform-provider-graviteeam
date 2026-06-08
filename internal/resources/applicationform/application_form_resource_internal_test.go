package applicationform

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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewApplicationFormResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_application_form"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewApplicationFormResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "application_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "template", true, false, false)
	assertBoolAttribute(t, resp.Schema.Attributes, "enabled", false, true, true)
	assertStringAttribute(t, resp.Schema.Attributes, "content", true, false, false)
}

func TestFormTemplateValidatorDescriptions(t *testing.T) {
	t.Parallel()

	validator := formTemplateValidator{}
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
	(&ApplicationFormResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationFormResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ApplicationFormResource{}
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

func TestBuildUpdateBodyPreservesCurrentAssets(t *testing.T) {
	t.Parallel()

	plan := ApplicationFormModel{
		Enabled: types.BoolValue(false),
		Content: types.StringValue("<html>updated</html>"),
	}
	current := map[string]interface{}{
		"assets": []interface{}{"logo.png"},
	}

	got := buildUpdateBody(plan, current)

	if got["enabled"] != false || got["content"] != "<html>updated</html>" {
		t.Fatalf("planned fields not applied: %#v", got)
	}
	if !reflect.DeepEqual(got["assets"], current["assets"]) {
		t.Fatalf("assets = %#v, want preserved %#v", got["assets"], current["assets"])
	}
}

func TestBuildUpdateBodyOmitsAssetsWhenCurrentHasNone(t *testing.T) {
	t.Parallel()

	plan := ApplicationFormModel{
		Enabled: types.BoolValue(true),
		Content: types.StringValue("<html>content</html>"),
	}

	got := buildUpdateBody(plan, map[string]interface{}{})

	if _, ok := got["assets"]; ok {
		t.Fatalf("assets should be omitted: %#v", got)
	}
}

func TestReadIntoModelMapsApplicationFormFields(t *testing.T) {
	t.Parallel()

	resource := &ApplicationFormResource{}
	model := ApplicationFormModel{}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":       "form-1",
		"template": "registration",
		"enabled":  true,
		"content":  "<html>registration</html>",
	})

	if model.ID.ValueString() != "form-1" ||
		model.Template.ValueString() != "REGISTRATION" ||
		!model.Enabled.ValueBool() ||
		model.Content.ValueString() != "<html>registration</html>" {
		t.Fatalf("model = %#v", model)
	}
}

func TestFormTemplateValidatorAcceptsKnownAndIgnoresUnknownValues(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]types.String{
		"valid":   types.StringValue("LOGIN"),
		"null":    types.StringNull(),
		"unknown": types.StringUnknown(),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var resp validator.StringResponse
			formTemplateValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: value}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
			}
		})
	}
}

func TestFormTemplateValidatorRejectsInvalidTemplate(t *testing.T) {
	t.Parallel()

	var resp validator.StringResponse
	formTemplateValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue("NOT_A_TEMPLATE")}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics for invalid template")
	}
}

func TestApplicationFormCRUDPreservesCurrentAssetsOnUpdate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	assets := []interface{}{"logo.png"}
	var bodies []map[string]interface{}
	var deletePaths []string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "form-123",
				"template": body["template"],
				"enabled":  body["enabled"],
				"content":  body["content"],
			})
		case http.MethodGet:
			if got, want := r.URL.Query().Get("template"), "LOGIN"; got != want {
				t.Fatalf("template query = %q, want %q", got, want)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "form-123",
				"template": "login",
				"enabled":  true,
				"content":  "<html>api</html>",
				"assets":   assets,
			})
		default:
			t.Fatalf("unexpected application form collection method %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "form-123",
				"template": "LOGIN",
				"enabled":  body["enabled"],
				"content":  body["content"],
			})
		case http.MethodDelete:
			deletePaths = append(deletePaths, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected application form item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := applicationFormPlan(t, schemaResp.Schema, ApplicationFormModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(true),
		Content:       types.StringValue("<html>create</html>"),
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

	updatePlan := applicationFormPlan(t, schemaResp.Schema, ApplicationFormModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(false),
		Content:       types.StringValue("<html>update</html>"),
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

	if got, want := bodies[0], (map[string]interface{}{"template": "LOGIN", "enabled": true, "content": "<html>create</html>"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("create body = %#v, want %#v", got, want)
	}
	if got, want := bodies[1]["assets"], assets; !reflect.DeepEqual(got, want) {
		t.Fatalf("update assets = %#v, want preserved %#v", got, want)
	}
	if got, want := deletePaths, []string{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms/form-123"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("delete paths = %#v, want %#v", got, want)
	}
}

func TestApplicationFormReadRemovesMissingForm(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ApplicationFormModel{
		ID:            types.StringValue("form-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(true),
		Content:       types.StringValue("<html>login</html>"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing application form to remove state, got %#v", readResp.State.Raw)
	}
}

func TestApplicationFormReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "read failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := applicationFormState(t, schemaResp.Schema, ApplicationFormModel{
		ID:            types.StringValue("form-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(true),
		Content:       types.StringValue("<html>login</html>"),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}
}

func TestApplicationFormCreateReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := applicationFormPlan(t, schemaResp.Schema, ApplicationFormModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(true),
		Content:       types.StringValue("<html>login</html>"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}
}

func TestApplicationFormUpdateAndDeleteReportRemoteErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("collection method = %s, want GET", r.Method)
		}
		http.Error(w, "read before update failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ApplicationFormModel{
		ID:            types.StringValue("form-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(true),
		Content:       types.StringValue("<html>login</html>"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	plan := applicationFormPlan(t, schemaResp.Schema, ApplicationFormModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(false),
		Content:       types.StringValue("<html>updated</html>"),
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

func TestApplicationFormDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: applicationFormState(t, schemaResp.Schema, baseApplicationFormModel()),
	}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestApplicationFormDeleteReportsInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ApplicationFormResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"domain_id":      tftypes.Number,
			"application_id": tftypes.String,
			"template":       tftypes.String,
			"enabled":        tftypes.Bool,
			"content":        tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":             tftypes.NewValue(tftypes.String, "form-123"),
			"domain_id":      tftypes.NewValue(tftypes.Number, 123),
			"application_id": tftypes.NewValue(tftypes.String, "app-123"),
			"template":       tftypes.NewValue(tftypes.String, "LOGIN"),
			"enabled":        tftypes.NewValue(tftypes.Bool, true),
			"content":        tftypes.NewValue(tftypes.String, "<html>login</html>"),
		},
	)

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestApplicationFormStopsOnInvalidPlanOrState(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewApplicationFormResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	invalidPlan := applicationFormInvalidPlan(schemaResp.Schema)
	invalidState := applicationFormInvalidState(schemaResp.Schema)
	validPlan := applicationFormPlan(t, schemaResp.Schema, baseApplicationFormModel())

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	(&ApplicationFormResource{}).Create(context.Background(), resource.CreateRequest{Plan: invalidPlan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	(&ApplicationFormResource{}).Read(context.Background(), resource.ReadRequest{State: invalidState}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	updatePlanResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	(&ApplicationFormResource{}).Update(context.Background(), resource.UpdateRequest{
		Plan:  invalidPlan,
		State: applicationFormState(t, schemaResp.Schema, baseApplicationFormModel()),
	}, updatePlanResp)
	if !updatePlanResp.Diagnostics.HasError() {
		t.Fatal("expected update plan diagnostics")
	}

	updateStateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	(&ApplicationFormResource{}).Update(context.Background(), resource.UpdateRequest{
		Plan:  validPlan,
		State: invalidState,
	}, updateStateResp)
	if !updateStateResp.Diagnostics.HasError() {
		t.Fatal("expected update state diagnostics")
	}
}

func TestApplicationFormUpdateReportsRemoteUpdateError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("collection method = %s, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "form-123",
			"template": "LOGIN",
			"enabled":  true,
			"content":  "<html>current</html>",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("item method = %s, want PUT", r.Method)
		}
		http.Error(w, "update failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := applicationFormState(t, schemaResp.Schema, ApplicationFormModel{
		ID:            types.StringValue("form-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(true),
		Content:       types.StringValue("<html>login</html>"),
	})
	plan := applicationFormPlan(t, schemaResp.Schema, ApplicationFormModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(false),
		Content:       types.StringValue("<html>updated</html>"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}
}

func TestApplicationFormImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&ApplicationFormResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/app-123",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestApplicationFormImportStateSetsAttributes(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&ApplicationFormResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: applicationFormState(t, schemaResp.Schema, baseApplicationFormModel())}

	(&ApplicationFormResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/app-123/LOGIN",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state ApplicationFormModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get import state: %#v", diags)
	}
	if got, want := state.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain_id = %q, want %q", got, want)
	}
	if got, want := state.ApplicationID.ValueString(), "app-123"; got != want {
		t.Fatalf("application_id = %q, want %q", got, want)
	}
	if got, want := state.Template.ValueString(), "LOGIN"; got != want {
		t.Fatalf("template = %q, want %q", got, want)
	}
}

func baseApplicationFormModel() ApplicationFormModel {
	return ApplicationFormModel{
		ID:            types.StringValue("form-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(true),
		Content:       types.StringValue("<html>login</html>"),
	}
}

func applicationFormPlan(t *testing.T, schema resourceschema.Schema, model ApplicationFormModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func applicationFormInvalidPlan(schema resourceschema.Schema) tfsdk.Plan {
	return tfsdk.Plan{Schema: schema, Raw: applicationFormInvalidRaw()}
}

func applicationFormState(t *testing.T, schema resourceschema.Schema, model ApplicationFormModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func applicationFormInvalidState(schema resourceschema.Schema) tfsdk.State {
	return tfsdk.State{Schema: schema, Raw: applicationFormInvalidRaw()}
}

func applicationFormInvalidRaw() tftypes.Value {
	return tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":             tftypes.String,
			"domain_id":      tftypes.Number,
			"application_id": tftypes.String,
			"template":       tftypes.String,
			"enabled":        tftypes.Bool,
			"content":        tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":             tftypes.NewValue(tftypes.String, "form-123"),
			"domain_id":      tftypes.NewValue(tftypes.Number, 123),
			"application_id": tftypes.NewValue(tftypes.String, "app-123"),
			"template":       tftypes.NewValue(tftypes.String, "LOGIN"),
			"enabled":        tftypes.NewValue(tftypes.Bool, true),
			"content":        tftypes.NewValue(tftypes.String, "<html>login</html>"),
		},
	)
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
