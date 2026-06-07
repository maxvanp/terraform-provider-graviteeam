package orgform

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
