package user

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewUserResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_user"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewUserResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "username"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{
		"email",
		"first_name",
		"last_name",
		"display_name",
		"force_reset_password",
		"enabled",
		"locked",
		"pre_registration",
		"reset_password",
		"reset_password_trigger",
		"registration_confirmation_trigger",
	} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "force_reset_password", "enabled", "locked", "pre_registration"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
	if attr := resp.Schema.Attributes["reset_password"]; attr == nil || !attr.IsSensitive() {
		t.Fatalf("reset_password should be sensitive")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&UserResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildMergedUpdateBodyPreservesUnmanagedCurrentFields(t *testing.T) {
	t.Parallel()

	current := map[string]interface{}{
		"id":                    "ignored",
		"accountNonExpired":     true,
		"accountNonLocked":      false,
		"additionalInformation": map[string]interface{}{"team": "platform"},
		"client":                "app",
		"createdAt":             "2026-01-01T00:00:00Z",
		"credentialsNonExpired": true,
		"displayName":           "Old Display",
		"email":                 "old@example.com",
		"enabled":               true,
		"externalId":            "external-1",
		"firstName":             "Old",
		"forceResetPassword":    false,
		"lastName":              "Name",
		"loggedAt":              "2026-01-02T00:00:00Z",
		"loginsCount":           float64(4),
		"preRegistration":       true,
		"preferredLanguage":     "fr",
		"registrationCompleted": false,
		"source":                "gravitee",
		"updatedAt":             "2026-01-03T00:00:00Z",
		"unexpected":            "must-not-leak",
	}
	plan := UserModel{
		Email:              types.StringValue("new@example.com"),
		FirstName:          types.StringValue("New"),
		LastName:           types.StringValue("Person"),
		DisplayName:        types.StringValue("New Display"),
		ForceResetPassword: types.BoolValue(true),
		PreRegistration:    types.BoolValue(false),
	}

	got := buildMergedUpdateBody(current, plan)

	for _, field := range []string{
		"accountNonExpired",
		"accountNonLocked",
		"additionalInformation",
		"client",
		"createdAt",
		"credentialsNonExpired",
		"enabled",
		"externalId",
		"loggedAt",
		"loginsCount",
		"preferredLanguage",
		"registrationCompleted",
		"source",
		"updatedAt",
	} {
		if !reflect.DeepEqual(got[field], current[field]) {
			t.Fatalf("%s = %#v, want preserved %#v", field, got[field], current[field])
		}
	}
	if got["email"] != "new@example.com" ||
		got["firstName"] != "New" ||
		got["lastName"] != "Person" ||
		got["displayName"] != "New Display" {
		t.Fatalf("planned profile fields not applied: %#v", got)
	}
	if got["forceResetPassword"] != true || got["preRegistration"] != false {
		t.Fatalf("planned booleans not applied: %#v", got)
	}
	if _, ok := got["unexpected"]; ok {
		t.Fatalf("unexpected field leaked into update body: %#v", got)
	}
}

func TestBuildMergedUpdateBodyKeepsCurrentOptionalFieldsWhenPlanNull(t *testing.T) {
	t.Parallel()

	current := map[string]interface{}{
		"email":       "old@example.com",
		"firstName":   "Old",
		"lastName":    "Name",
		"displayName": "Old Display",
	}
	plan := UserModel{
		Email:              types.StringNull(),
		FirstName:          types.StringUnknown(),
		LastName:           types.StringNull(),
		DisplayName:        types.StringUnknown(),
		ForceResetPassword: types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
	}

	got := buildMergedUpdateBody(current, plan)

	for _, field := range []string{"email", "firstName", "lastName", "displayName"} {
		if got[field] != current[field] {
			t.Fatalf("%s = %#v, want preserved %#v", field, got[field], current[field])
		}
	}
}

func TestCopyUserUpdateFieldsOnlyCopiesAllowedFields(t *testing.T) {
	t.Parallel()

	current := map[string]interface{}{
		"email":      "alice@example.com",
		"enabled":    true,
		"unexpected": "ignored",
	}

	got := copyUserUpdateFields(current)

	if !reflect.DeepEqual(got, map[string]interface{}{"email": "alice@example.com", "enabled": true}) {
		t.Fatalf("copied fields = %#v", got)
	}
}

func TestShouldTrigger(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		plan types.String
		prev types.String
		want bool
	}{
		"null plan does not trigger":         {types.StringNull(), types.StringValue("old"), false},
		"unknown plan does not trigger":      {types.StringUnknown(), types.StringValue("old"), false},
		"new trigger runs":                   {types.StringValue("new"), types.StringNull(), true},
		"changed trigger runs":               {types.StringValue("new"), types.StringValue("old"), true},
		"unchanged trigger does not run":     {types.StringValue("same"), types.StringValue("same"), false},
		"reset wrapper delegates to trigger": {types.StringValue("new"), types.StringValue("old"), true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := shouldTrigger(test.plan, test.prev)
			if name == "reset wrapper delegates to trigger" {
				got = shouldResetPassword(test.plan, test.prev)
			}
			if got != test.want {
				t.Fatalf("trigger decision = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReadIntoModelMapsAPIFieldsAndLockState(t *testing.T) {
	t.Parallel()

	resource := &UserResource{}
	model := UserModel{
		Email:       types.StringValue("previous@example.com"),
		FirstName:   types.StringValue("Previous"),
		LastName:    types.StringValue("Name"),
		DisplayName: types.StringValue("Previous Name"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":                 "user-1",
		"username":           "alice",
		"email":              "alice@example.com",
		"firstName":          "Alice",
		"lastName":           "Liddell",
		"displayName":        "Alice Liddell",
		"forceResetPassword": true,
		"enabled":            true,
		"accountNonLocked":   false,
		"preRegistration":    false,
	})

	if model.ID.ValueString() != "user-1" ||
		model.Username.ValueString() != "alice" ||
		model.Email.ValueString() != "alice@example.com" ||
		model.FirstName.ValueString() != "Alice" ||
		model.LastName.ValueString() != "Liddell" ||
		model.DisplayName.ValueString() != "Alice Liddell" {
		t.Fatalf("string fields not mapped: %#v", model)
	}
	if !model.ForceResetPassword.ValueBool() || !model.Enabled.ValueBool() || !model.Locked.ValueBool() || model.PreRegistration.ValueBool() {
		t.Fatalf("boolean fields not mapped: %#v", model)
	}
}

func TestReadIntoModelClearsMissingOptionalStrings(t *testing.T) {
	t.Parallel()

	resource := &UserResource{}
	model := UserModel{
		Email:       types.StringValue("previous@example.com"),
		FirstName:   types.StringValue("Previous"),
		LastName:    types.StringValue("Name"),
		DisplayName: types.StringValue("Previous Name"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":       "user-1",
		"username": "alice",
	})

	if !model.Email.IsNull() || !model.FirstName.IsNull() || !model.LastName.IsNull() || !model.DisplayName.IsNull() {
		t.Fatalf("optional strings were not cleared: %#v", model)
	}
}
