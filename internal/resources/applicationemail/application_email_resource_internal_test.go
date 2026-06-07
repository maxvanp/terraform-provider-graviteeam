package applicationemail

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
