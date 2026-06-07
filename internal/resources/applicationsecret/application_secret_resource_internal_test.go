package applicationsecret

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
