package form

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

func TestFormMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewFormResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_form" {
		t.Fatalf("type name = %q, want graviteeam_form", resp.TypeName)
	}
}

func TestFormSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewFormResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "template", "content"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["application_id"]; !attr.IsOptional() {
		t.Fatal("application_id should be optional")
	}
	if attr := resp.Schema.Attributes["enabled"]; !attr.IsOptional() || !attr.IsComputed() {
		t.Fatalf("enabled should be optional+computed, got optional=%t computed=%t", attr.IsOptional(), attr.IsComputed())
	}
	if attr := resp.Schema.Attributes["id"]; !attr.IsComputed() {
		t.Fatal("id should be computed")
	}
}

func TestFormConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &FormResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestFormConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&FormResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestBuildUpdateBodyPreservesCurrentAssets(t *testing.T) {
	t.Parallel()

	plan := FormModel{
		Enabled: types.BoolValue(false),
		Content: types.StringValue("<html>updated</html>"),
	}
	current := map[string]interface{}{
		"assets": map[string]interface{}{"logo.png": "base64"},
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

	plan := FormModel{
		Enabled: types.BoolValue(true),
		Content: types.StringValue("<html>content</html>"),
	}

	got := buildUpdateBody(plan, map[string]interface{}{})

	if _, ok := got["assets"]; ok {
		t.Fatalf("assets should be omitted: %#v", got)
	}
}

func TestReadIntoModelMapsFormFields(t *testing.T) {
	t.Parallel()

	resource := &FormResource{}
	model := FormModel{}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":       "form-1",
		"template": "login",
		"enabled":  false,
		"content":  "<html>login</html>",
	})

	if model.ID.ValueString() != "form-1" ||
		model.Template.ValueString() != "LOGIN" ||
		model.Enabled.ValueBool() ||
		model.Content.ValueString() != "<html>login</html>" {
		t.Fatalf("model = %#v", model)
	}
}

func TestTemplateValidatorAcceptsKnownAndIgnoresUnknownValues(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]types.String{
		"valid":   types.StringValue("LOGIN"),
		"null":    types.StringNull(),
		"unknown": types.StringUnknown(),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var resp validator.StringResponse
			templateValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: value}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %#v", resp.Diagnostics)
			}
		})
	}
}

func TestTemplateValidatorRejectsInvalidTemplate(t *testing.T) {
	t.Parallel()

	var resp validator.StringResponse
	templateValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue("NOT_A_TEMPLATE")}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics for invalid template")
	}
}

func TestTemplateValidatorDescriptions(t *testing.T) {
	t.Parallel()

	validator := templateValidator{}
	if validator.Description(context.Background()) == "" {
		t.Fatal("expected non-empty description")
	}
	if validator.MarkdownDescription(context.Background()) == "" {
		t.Fatal("expected non-empty markdown description")
	}
}

func TestFormCRUDPreservesCurrentAssetsOnUpdate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	assets := map[string]interface{}{"logo.png": "base64"}
	var bodies []map[string]interface{}
	var deletePaths []string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms", func(w http.ResponseWriter, r *http.Request) {
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
			t.Fatalf("unexpected form collection method %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
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
			t.Fatalf("unexpected form item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := formPlan(t, schemaResp.Schema, FormModel{
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>create</html>"),
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

	updatePlan := formPlan(t, schemaResp.Schema, FormModel{
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(false),
		Content:  types.StringValue("<html>update</html>"),
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
	if got, want := deletePaths, []string{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms/form-123"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("delete paths = %#v, want %#v", got, want)
	}
}

func TestFormReadRemovesMissingForm(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &FormModel{
		ID:       types.StringValue("form-123"),
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing form to remove state, got %#v", readResp.State.Raw)
	}
}

func TestFormReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "forms failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := formState(t, schemaResp.Schema, FormModel{
		ID:       types.StringValue("form-123"),
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}
}

func TestFormDeleteIgnoresMissingForm(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := formState(t, schemaResp.Schema, FormModel{
		ID:       types.StringValue("form-123"),
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	})

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestFormDeleteUsesApplicationScopedPath(t *testing.T) {
	var deletePaths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		deletePaths = append(deletePaths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := formState(t, schemaResp.Schema, FormModel{
		ID:            types.StringValue("form-123"),
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Template:      types.StringValue("LOGIN"),
		Enabled:       types.BoolValue(true),
		Content:       types.StringValue("<html>login</html>"),
	})

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
	if got, want := deletePaths, []string{"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/forms/form-123"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("delete paths = %#v, want %#v", got, want)
	}
}

func TestFormCreateReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := formPlan(t, schemaResp.Schema, FormModel{
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}
}

func TestFormUpdateAndDeleteReportRemoteErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("collection method = %s, want GET", r.Method)
		}
		http.Error(w, "read before update failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &FormModel{
		ID:       types.StringValue("form-123"),
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	plan := formPlan(t, schemaResp.Schema, FormModel{
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(false),
		Content:  types.StringValue("<html>updated</html>"),
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

func TestFormUpdateReportsRemoteUpdateError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("item method = %s, want PUT", r.Method)
		}
		http.Error(w, "update failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := formState(t, schemaResp.Schema, FormModel{
		ID:       types.StringValue("form-123"),
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	})
	plan := formPlan(t, schemaResp.Schema, FormModel{
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(false),
		Content:  types.StringValue("<html>updated</html>"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}
}

func TestFormCRUDReportsInvalidPlanAndStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &FormResource{}
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
			"application_id": tftypes.NewValue(tftypes.String, nil),
			"template":       tftypes.NewValue(tftypes.String, "LOGIN"),
			"enabled":        tftypes.NewValue(tftypes.Bool, true),
			"content":        tftypes.NewValue(tftypes.String, "<html>login</html>"),
		},
	)
	valid := FormModel{
		ID:       types.StringValue("form-123"),
		DomainID: types.StringValue("domain-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	}

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

	invalidPlanResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		State: formState(t, schemaResp.Schema, valid),
	}, invalidPlanResp)
	if !invalidPlanResp.Diagnostics.HasError() {
		t.Fatal("expected invalid plan diagnostics")
	}

	invalidStateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  formPlan(t, schemaResp.Schema, valid),
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, invalidStateResp)
	if !invalidStateResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestFormImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&FormResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "invalid",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestFormImportSetsDomainTemplateAndOptionalApplication(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		id              string
		wantDomain      string
		wantApplication types.String
		wantTemplate    string
	}{
		"domain": {
			id:              "domain-123/LOGIN",
			wantDomain:      "domain-123",
			wantApplication: types.StringNull(),
			wantTemplate:    "LOGIN",
		},
		"application": {
			id:              "domain-123/app-123/LOGIN",
			wantDomain:      "domain-123",
			wantApplication: types.StringValue("app-123"),
			wantTemplate:    "LOGIN",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var schemaResp resource.SchemaResponse
			NewFormResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := formState(t, schemaResp.Schema, FormModel{
				ID:            types.StringValue("form-123"),
				DomainID:      types.StringValue("placeholder-domain"),
				ApplicationID: types.StringNull(),
				Template:      types.StringValue("ERROR"),
				Enabled:       types.BoolValue(true),
				Content:       types.StringValue("<html>login</html>"),
			})
			importResp := &resource.ImportStateResponse{State: state}

			NewFormResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
				ID: test.id,
			}, importResp)

			if importResp.Diagnostics.HasError() {
				t.Fatalf("import diagnostics: %#v", importResp.Diagnostics)
			}
			var imported FormModel
			if diags := importResp.State.Get(context.Background(), &imported); diags.HasError() {
				t.Fatalf("get imported state: %#v", diags)
			}
			if imported.DomainID.ValueString() != test.wantDomain ||
				!imported.ApplicationID.Equal(test.wantApplication) ||
				imported.Template.ValueString() != test.wantTemplate {
				t.Fatalf("imported = %#v", imported)
			}
		})
	}
}

func formPlan(t *testing.T, schema resourceschema.Schema, model FormModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func formState(t *testing.T, schema resourceschema.Schema, model FormModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
