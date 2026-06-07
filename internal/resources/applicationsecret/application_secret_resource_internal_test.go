package applicationsecret

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
	NewApplicationSecretResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_application_secret"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewApplicationSecretResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "application_id", true, false, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "renew_trigger", false, true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "secret", false, false, true, true)
	assertStringAttribute(t, resp.Schema.Attributes, "settings_id", false, false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "expires_at", false, false, true, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationSecretResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestReadIntoModelUsesReturnedSecretAndMetadata(t *testing.T) {
	model := ApplicationSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":         "secret-id",
		"name":       "client secret",
		"secret":     "clear-secret",
		"settingsId": "settings-id",
		"expiresAt":  "2026-06-07T12:00:00Z",
	}, types.StringValue("old-secret"))

	if model.ID.ValueString() != "secret-id" {
		t.Fatalf("id = %q, want secret-id", model.ID.ValueString())
	}
	if model.Name.ValueString() != "client secret" {
		t.Fatalf("name = %q, want client secret", model.Name.ValueString())
	}
	if model.Secret.ValueString() != "clear-secret" {
		t.Fatalf("secret = %q, want clear-secret", model.Secret.ValueString())
	}
	if model.SettingsID.ValueString() != "settings-id" {
		t.Fatalf("settings_id = %q, want settings-id", model.SettingsID.ValueString())
	}
	if model.ExpiresAt.ValueString() != "2026-06-07T12:00:00Z" {
		t.Fatalf("expires_at = %q, want timestamp", model.ExpiresAt.ValueString())
	}
}

func TestReadIntoModelPreservesExistingSecretWhenAPIOmitsClearValue(t *testing.T) {
	model := ApplicationSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":     "secret-id",
		"name":   "client secret",
		"secret": "",
	}, types.StringValue("preserved-secret"))

	if model.Secret.ValueString() != "preserved-secret" {
		t.Fatalf("secret = %q, want preserved-secret", model.Secret.ValueString())
	}
	if !model.SettingsID.IsNull() {
		t.Fatalf("settings_id = %#v, want null", model.SettingsID)
	}
	if !model.ExpiresAt.IsNull() {
		t.Fatalf("expires_at = %#v, want null", model.ExpiresAt)
	}
}

func TestReadIntoModelNullsSecretWhenNoClearOrPreservedValue(t *testing.T) {
	model := ApplicationSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":   "secret-id",
		"name": "client secret",
	}, types.StringUnknown())

	if !model.Secret.IsNull() {
		t.Fatalf("secret = %#v, want null", model.Secret)
	}
}

func TestShouldRenew(t *testing.T) {
	tests := []struct {
		name  string
		plan  types.String
		state types.String
		want  bool
	}{
		{name: "plan null", plan: types.StringNull(), state: types.StringValue("old"), want: false},
		{name: "plan unknown", plan: types.StringUnknown(), state: types.StringValue("old"), want: false},
		{name: "state null", plan: types.StringValue("new"), state: types.StringNull(), want: true},
		{name: "state unknown", plan: types.StringValue("new"), state: types.StringUnknown(), want: true},
		{name: "same value", plan: types.StringValue("same"), state: types.StringValue("same"), want: false},
		{name: "changed value", plan: types.StringValue("new"), state: types.StringValue("old"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRenew(tt.plan, tt.state); got != tt.want {
				t.Fatalf("shouldRenew() = %t, want %t", got, tt.want)
			}
		})
	}
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed, sensitive bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed || attr.Sensitive != sensitive {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t sensitive:%t, want required:%t optional:%t computed:%t sensitive:%t",
			name, attr.Required, attr.Optional, attr.Computed, attr.Sensitive, required, optional, computed, sensitive)
	}
}
