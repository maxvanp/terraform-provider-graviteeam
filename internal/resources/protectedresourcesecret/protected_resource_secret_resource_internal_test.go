package protectedresourcesecret

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		id                      string
		wantDomainID            string
		wantProtectedResourceID string
		wantSecretID            string
		wantOK                  bool
	}{
		{
			name:                    "valid",
			id:                      "domain-1/protected-resource-1/secret-1",
			wantDomainID:            "domain-1",
			wantProtectedResourceID: "protected-resource-1",
			wantSecretID:            "secret-1",
			wantOK:                  true,
		},
		{
			name:   "missing part",
			id:     "domain-1/protected-resource-1",
			wantOK: false,
		},
		{
			name:   "extra part",
			id:     "domain-1/protected-resource-1/secret-1/extra",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotProtectedResourceID, gotSecretID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotProtectedResourceID != tt.wantProtectedResourceID {
				t.Fatalf("protected resource ID = %q, want %q", gotProtectedResourceID, tt.wantProtectedResourceID)
			}
			if gotSecretID != tt.wantSecretID {
				t.Fatalf("secret ID = %q, want %q", gotSecretID, tt.wantSecretID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := ProtectedResourceSecretModel{
		Name: types.StringValue("client-secret"),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name": "client-secret",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelUsesReturnedSecret(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":         "secret-1",
		"name":       "client-secret",
		"secret":     "clear-secret",
		"settingsId": "settings-1",
		"expiresAt":  "2026-06-07T12:00:00Z",
	}, types.StringValue("old-secret"))

	if got, want := model.ID.ValueString(), "secret-1"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.Name.ValueString(), "client-secret"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Secret.ValueString(), "clear-secret"; got != want {
		t.Fatalf("secret = %q, want %q", got, want)
	}
	if got, want := model.SettingsID.ValueString(), "settings-1"; got != want {
		t.Fatalf("settings ID = %q, want %q", got, want)
	}
	if got, want := model.ExpiresAt.ValueString(), "2026-06-07T12:00:00Z"; got != want {
		t.Fatalf("expires at = %q, want %q", got, want)
	}
}

func TestReadIntoModelPreservesExistingSecretWhenAPIOmitsClearValue(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":     "secret-1",
		"name":   "client-secret",
		"secret": "",
	}, types.StringValue("preserved-secret"))

	if got, want := model.Secret.ValueString(), "preserved-secret"; got != want {
		t.Fatalf("secret = %q, want preserved %q", got, want)
	}
	if !model.SettingsID.IsNull() {
		t.Fatalf("settings ID should be null, got %q", model.SettingsID.ValueString())
	}
	if !model.ExpiresAt.IsNull() {
		t.Fatalf("expires at should be null, got %q", model.ExpiresAt.ValueString())
	}
}

func TestReadIntoModelSetsNullSecretWhenNoPreservedValue(t *testing.T) {
	t.Parallel()

	model := ProtectedResourceSecretModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":   "secret-1",
		"name": "client-secret",
	}, types.StringUnknown())

	if !model.Secret.IsNull() {
		t.Fatalf("secret should be null, got %q", model.Secret.ValueString())
	}
}

func TestShouldRenew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		planTrigger  types.String
		stateTrigger types.String
		want         bool
	}{
		{
			name:         "null plan does not renew",
			planTrigger:  types.StringNull(),
			stateTrigger: types.StringValue("old"),
			want:         false,
		},
		{
			name:         "unknown plan does not renew",
			planTrigger:  types.StringUnknown(),
			stateTrigger: types.StringValue("old"),
			want:         false,
		},
		{
			name:         "first configured trigger renews",
			planTrigger:  types.StringValue("new"),
			stateTrigger: types.StringNull(),
			want:         true,
		},
		{
			name:         "changed trigger renews",
			planTrigger:  types.StringValue("new"),
			stateTrigger: types.StringValue("old"),
			want:         true,
		},
		{
			name:         "same trigger does not renew",
			planTrigger:  types.StringValue("same"),
			stateTrigger: types.StringValue("same"),
			want:         false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldRenew(tt.planTrigger, tt.stateTrigger); got != tt.want {
				t.Fatalf("shouldRenew = %t, want %t", got, tt.want)
			}
		})
	}
}
