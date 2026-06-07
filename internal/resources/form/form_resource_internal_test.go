package form

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
