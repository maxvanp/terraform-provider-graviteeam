package orgform

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

func orgFormPlan(t *testing.T, schema resourceschema.Schema, model OrgFormModel) tfsdk.Plan {
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
