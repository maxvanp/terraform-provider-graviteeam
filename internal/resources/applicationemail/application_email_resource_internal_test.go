package applicationemail

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
