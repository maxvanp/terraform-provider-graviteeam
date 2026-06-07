package alertnotifier

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildCreateBodyIncludesImmutableAndMutableFields(t *testing.T) {
	model := AlertNotifierModel{
		Name:          types.StringValue("webhook"),
		Type:          types.StringValue("webhook-notifier"),
		Configuration: types.StringValue(`{"url":"https://example.test","method":"POST"}`),
		Enabled:       types.BoolValue(true),
	}

	body := buildCreateBody(model)

	if body["name"] != "webhook" {
		t.Fatalf("name = %#v, want webhook", body["name"])
	}
	if body["type"] != "webhook-notifier" {
		t.Fatalf("type = %#v, want webhook-notifier", body["type"])
	}
	if body["configuration"] != `{"url":"https://example.test","method":"POST"}` {
		t.Fatalf("configuration = %#v, want original JSON", body["configuration"])
	}
	if body["enabled"] != true {
		t.Fatalf("enabled = %#v, want true", body["enabled"])
	}
}

func TestBuildPatchBodyOmitsImmutableType(t *testing.T) {
	model := AlertNotifierModel{
		Name:          types.StringValue("webhook"),
		Type:          types.StringValue("webhook-notifier"),
		Configuration: types.StringValue(`{"url":"https://example.test"}`),
		Enabled:       types.BoolValue(false),
	}

	body := buildPatchBody(model)

	if _, ok := body["type"]; ok {
		t.Fatalf("type = %#v, want omitted for PATCH", body["type"])
	}
	if body["name"] != "webhook" {
		t.Fatalf("name = %#v, want webhook", body["name"])
	}
	if body["configuration"] != `{"url":"https://example.test"}` {
		t.Fatalf("configuration = %#v, want original JSON", body["configuration"])
	}
	if body["enabled"] != false {
		t.Fatalf("enabled = %#v, want false", body["enabled"])
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	model := AlertNotifierModel{
		Name:          types.StringValue("old"),
		Type:          types.StringValue("webhook-notifier"),
		Configuration: types.StringValue(`{"secret":"plain"}`),
		Enabled:       types.BoolValue(false),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new",
		"type":          "webhook-notifier",
		"configuration": `{"secret":"***"}`,
		"enabled":       true,
	})

	if model.Name.ValueString() != "new" {
		t.Fatalf("name = %q, want new", model.Name.ValueString())
	}
	if model.Type.ValueString() != "webhook-notifier" {
		t.Fatalf("type = %q, want webhook-notifier", model.Type.ValueString())
	}
	if model.Configuration.ValueString() != `{"secret":"plain"}` {
		t.Fatalf("configuration = %q, want preserved unmasked state", model.Configuration.ValueString())
	}
	if !model.Enabled.ValueBool() {
		t.Fatal("enabled = false, want true")
	}
}
