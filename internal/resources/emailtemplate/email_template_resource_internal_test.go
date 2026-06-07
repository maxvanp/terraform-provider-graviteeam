package emailtemplate

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
