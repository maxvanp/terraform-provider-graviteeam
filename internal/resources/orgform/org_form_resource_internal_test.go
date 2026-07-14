package orgform

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
	NewOrgFormResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_form"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgFormResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "template", true, false, false)
	assertBoolAttribute(t, resp.Schema.Attributes, "enabled", false, true, true)
	assertStringAttribute(t, resp.Schema.Attributes, "content", true, false, false)
}

func TestTemplateValidatorDescriptions(t *testing.T) {
	t.Parallel()

	validator := templateValidator{}
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
	(&OrgFormResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgFormResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgFormResource{}
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

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := OrgFormModel{
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	}

	got := buildCreateBody(plan)
	want := map[string]interface{}{
		"template": "LOGIN",
		"enabled":  true,
		"content":  "<html>login</html>",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyPreservesAssets(t *testing.T) {
	t.Parallel()

	assets := []interface{}{
		map[string]interface{}{"name": "logo.png", "content": "base64"},
	}
	plan := OrgFormModel{
		Enabled: types.BoolValue(false),
		Content: types.StringValue("<html>updated</html>"),
	}

	got := buildUpdateBody(plan, map[string]interface{}{"assets": assets})
	want := map[string]interface{}{
		"enabled": false,
		"content": "<html>updated</html>",
		"assets":  assets,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyOmitsMissingAssets(t *testing.T) {
	t.Parallel()

	plan := OrgFormModel{
		Enabled: types.BoolValue(true),
		Content: types.StringValue("<html>updated</html>"),
	}

	got := buildUpdateBody(plan, map[string]interface{}{})
	want := map[string]interface{}{
		"enabled": true,
		"content": "<html>updated</html>",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelMapsOrgForm(t *testing.T) {
	t.Parallel()

	model := OrgFormModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":       "form-id",
		"template": "login",
		"enabled":  true,
		"content":  "<html>login</html>",
	})

	if got, want := model.ID.ValueString(), "form-id"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.Template.ValueString(), "LOGIN"; got != want {
		t.Fatalf("template = %q, want %q", got, want)
	}
	if !model.Enabled.ValueBool() {
		t.Fatalf("enabled should be true")
	}
	if got, want := model.Content.ValueString(), "<html>login</html>"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
}

func TestTemplateValidatorAcceptsKnownTemplatesCaseInsensitively(t *testing.T) {
	t.Parallel()

	for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
		resp := validator.StringResponse{}
		templateValidator{}.ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: value,
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("null/unknown template diagnostics: %v", resp.Diagnostics)
		}
	}

	resp := validator.StringResponse{}
	templateValidator{}.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringValue("login"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected lower-case template to be accepted, got %v", resp.Diagnostics)
	}
}

func TestTemplateValidatorRejectsUnknownTemplate(t *testing.T) {
	t.Parallel()

	resp := validator.StringResponse{}
	templateValidator{}.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringValue("NOT_A_TEMPLATE"),
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected unknown template to be rejected")
	}
}

func TestOrgFormCRUDPreservesCurrentAssetsOnUpdate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	assets := []interface{}{map[string]interface{}{"name": "logo.png", "content": "base64"}}
	var bodies []map[string]interface{}
	var deletePaths []string
	mux.HandleFunc("/management/organizations/DEFAULT/forms", func(w http.ResponseWriter, r *http.Request) {
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
			t.Fatalf("unexpected org form collection method %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
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
			t.Fatalf("unexpected org form item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgFormPlan(t, schemaResp.Schema, OrgFormModel{
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

	updatePlan := orgFormPlan(t, schemaResp.Schema, OrgFormModel{
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
	if got, want := deletePaths, []string{"/management/organizations/DEFAULT/forms/form-123"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("delete paths = %#v, want %#v", got, want)
	}
}

func TestOrgFormReadRemovesMissingFormAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing form", statusCode: http.StatusNotFound, wantRemove: true},
		{name: "server error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/forms", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				if got, want := r.URL.Query().Get("template"), "LOGIN"; got != want {
					t.Fatalf("template query = %q, want %q", got, want)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &OrgFormResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := orgFormState(t, schemaResp.Schema, OrgFormModel{
				ID:       types.StringValue("form-123"),
				Template: types.StringValue("LOGIN"),
				Enabled:  types.BoolValue(true),
				Content:  types.StringValue("<html>login</html>"),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing form to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestOrgFormReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/forms", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			http.Error(w, "create failed", http.StatusInternalServerError)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "form-123",
				"template": "LOGIN",
				"enabled":  true,
				"content":  "<html>current</html>",
			})
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &OrgFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := orgFormPlan(t, schemaResp.Schema, OrgFormModel{
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := orgFormState(t, schemaResp.Schema, OrgFormModel{
		ID:       types.StringValue("form-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
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

func TestOrgFormDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/forms/form-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: orgFormState(t, schemaResp.Schema, baseOrgFormModel()),
	}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestOrgFormDeleteReportsInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgFormResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":       tftypes.Number,
			"template": tftypes.String,
			"enabled":  tftypes.Bool,
			"content":  tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":       tftypes.NewValue(tftypes.Number, 123),
			"template": tftypes.NewValue(tftypes.String, "LOGIN"),
			"enabled":  tftypes.NewValue(tftypes.Bool, true),
			"content":  tftypes.NewValue(tftypes.String, "<html>login</html>"),
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

func TestOrgFormStopsOnInvalidPlanOrState(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewOrgFormResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	invalidPlan := orgFormInvalidPlan(schemaResp.Schema)
	invalidState := orgFormInvalidState(schemaResp.Schema)
	validPlan := orgFormPlan(t, schemaResp.Schema, baseOrgFormModel())

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	(&OrgFormResource{}).Create(context.Background(), resource.CreateRequest{Plan: invalidPlan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	(&OrgFormResource{}).Read(context.Background(), resource.ReadRequest{State: invalidState}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	updatePlanResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	(&OrgFormResource{}).Update(context.Background(), resource.UpdateRequest{
		Plan:  invalidPlan,
		State: orgFormState(t, schemaResp.Schema, baseOrgFormModel()),
	}, updatePlanResp)
	if !updatePlanResp.Diagnostics.HasError() {
		t.Fatal("expected update plan diagnostics")
	}

	updateStateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	(&OrgFormResource{}).Update(context.Background(), resource.UpdateRequest{
		Plan:  validPlan,
		State: invalidState,
	}, updateStateResp)
	if !updateStateResp.Diagnostics.HasError() {
		t.Fatal("expected update state diagnostics")
	}
}

func TestOrgFormUpdateReportsPreReadError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/forms", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "pre-read failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgFormResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := orgFormPlan(t, schemaResp.Schema, OrgFormModel{
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	})
	state := orgFormState(t, schemaResp.Schema, OrgFormModel{
		ID:       types.StringValue("form-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	})

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update pre-read diagnostics")
	}
}

func TestOrgFormImportStateSetsTemplate(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&OrgFormResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: orgFormState(t, schemaResp.Schema, baseOrgFormModel())}

	(&OrgFormResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "LOGIN",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state OrgFormModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get import state: %#v", diags)
	}
	if got, want := state.Template.ValueString(), "LOGIN"; got != want {
		t.Fatalf("template = %q, want %q", got, want)
	}
}

func baseOrgFormModel() OrgFormModel {
	return OrgFormModel{
		ID:       types.StringValue("form-123"),
		Template: types.StringValue("LOGIN"),
		Enabled:  types.BoolValue(true),
		Content:  types.StringValue("<html>login</html>"),
	}
}

func orgFormPlan(t *testing.T, schema resourceschema.Schema, model OrgFormModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func orgFormInvalidPlan(schema resourceschema.Schema) tfsdk.Plan {
	return tfsdk.Plan{Schema: schema, Raw: orgFormInvalidRaw()}
}

func orgFormState(t *testing.T, schema resourceschema.Schema, model OrgFormModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func orgFormInvalidState(schema resourceschema.Schema) tfsdk.State {
	return tfsdk.State{Schema: schema, Raw: orgFormInvalidRaw()}
}

func orgFormInvalidRaw() tftypes.Value {
	return tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":       tftypes.Number,
			"template": tftypes.String,
			"enabled":  tftypes.Bool,
			"content":  tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":       tftypes.NewValue(tftypes.Number, 123),
			"template": tftypes.NewValue(tftypes.String, "LOGIN"),
			"enabled":  tftypes.NewValue(tftypes.Bool, true),
			"content":  tftypes.NewValue(tftypes.String, "<html>login</html>"),
		},
	)
}

func assertStringAttribute(t *testing.T, attrs map[string]resourceschema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want resourceschema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}

func assertBoolAttribute(t *testing.T, attrs map[string]resourceschema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(resourceschema.BoolAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want resourceschema.BoolAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
