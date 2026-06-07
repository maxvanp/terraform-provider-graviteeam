package applicationform

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
