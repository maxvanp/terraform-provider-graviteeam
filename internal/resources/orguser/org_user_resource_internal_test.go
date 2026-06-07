package orguser

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildCreateBodyIncludesRequiredAndOptionalFields(t *testing.T) {
	t.Parallel()

	model := OrgUserModel{
		Username:           types.StringValue("alice"),
		Password:           types.StringValue("secret"),
		Email:              types.StringValue("alice@example.com"),
		FirstName:          types.StringValue("Alice"),
		LastName:           types.StringValue("Liddell"),
		ForceResetPassword: types.BoolValue(true),
		PreRegistration:    types.BoolValue(false),
	}

	got := buildCreateBody(model)
	want := map[string]interface{}{
		"username":           "alice",
		"password":           "secret",
		"email":              "alice@example.com",
		"firstName":          "Alice",
		"lastName":           "Liddell",
		"forceResetPassword": true,
		"preRegistration":    false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("create body = %#v, want %#v", got, want)
	}
}

func TestBuildCreateBodyOmitsUnknownPasswordAndNullProfileFields(t *testing.T) {
	t.Parallel()

	model := OrgUserModel{
		Username:           types.StringValue("alice"),
		Password:           types.StringUnknown(),
		Email:              types.StringNull(),
		FirstName:          types.StringNull(),
		LastName:           types.StringNull(),
		ForceResetPassword: types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
	}

	got := buildCreateBody(model)
	want := map[string]interface{}{
		"username":           "alice",
		"forceResetPassword": false,
		"preRegistration":    true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("create body = %#v, want %#v", got, want)
	}
}

func TestBuildMergedUpdateBodyPreservesUnmanagedCurrentFields(t *testing.T) {
	t.Parallel()

	current := map[string]interface{}{
		"id":                    "ignored",
		"accountNonExpired":     true,
		"accountNonLocked":      true,
		"additionalInformation": map[string]interface{}{"department": "platform"},
		"client":                "api",
		"createdAt":             "2026-01-01T00:00:00Z",
		"credentialsNonExpired": true,
		"displayName":           "Old Name",
		"email":                 "old@example.com",
		"enabled":               true,
		"externalId":            "external-1",
		"firstName":             "Old",
		"forceResetPassword":    false,
		"lastName":              "Name",
		"loggedAt":              "2026-01-02T00:00:00Z",
		"loginsCount":           float64(3),
		"preRegistration":       true,
		"preferredLanguage":     "fr",
		"registrationCompleted": false,
		"source":                "gravitee",
		"updatedAt":             "2026-01-03T00:00:00Z",
		"unexpected":            "must-not-leak",
	}
	plan := OrgUserModel{
		Email:              types.StringValue("new@example.com"),
		FirstName:          types.StringValue("New"),
		LastName:           types.StringValue("Person"),
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
		"displayName",
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
	if got["email"] != "new@example.com" || got["firstName"] != "New" || got["lastName"] != "Person" {
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
		"email":     "old@example.com",
		"firstName": "Old",
		"lastName":  "Name",
	}
	plan := OrgUserModel{
		Email:              types.StringNull(),
		FirstName:          types.StringUnknown(),
		LastName:           types.StringNull(),
		ForceResetPassword: types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
	}

	got := buildMergedUpdateBody(current, plan)

	if got["email"] != current["email"] || got["firstName"] != current["firstName"] || got["lastName"] != current["lastName"] {
		t.Fatalf("current optional fields were not preserved: %#v", got)
	}
}

func TestShouldResetPassword(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		plan types.String
		prev types.String
		want bool
	}{
		"null plan does not reset":         {types.StringNull(), types.StringValue("old"), false},
		"unknown plan does not reset":      {types.StringUnknown(), types.StringValue("old"), false},
		"new trigger resets":               {types.StringValue("new"), types.StringNull(), true},
		"changed trigger resets":           {types.StringValue("new"), types.StringValue("old"), true},
		"unchanged trigger does not reset": {types.StringValue("same"), types.StringValue("same"), false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := shouldResetPassword(test.plan, test.prev); got != test.want {
				t.Fatalf("shouldResetPassword = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReadIntoModelMapsAPIFieldsAndClearsMissingOptionalStrings(t *testing.T) {
	t.Parallel()

	model := OrgUserModel{
		Email:     types.StringValue("previous@example.com"),
		FirstName: types.StringValue("Previous"),
		LastName:  types.StringValue("Name"),
	}
	readIntoModel(&model, map[string]interface{}{
		"id":                 "user-1",
		"username":           "alice",
		"forceResetPassword": true,
		"enabled":            false,
		"preRegistration":    true,
	})

	if model.ID.ValueString() != "user-1" || model.Username.ValueString() != "alice" {
		t.Fatalf("identity fields not mapped: %#v", model)
	}
	if !model.Email.IsNull() || !model.FirstName.IsNull() || !model.LastName.IsNull() {
		t.Fatalf("optional strings were not cleared: %#v", model)
	}
	if !model.ForceResetPassword.ValueBool() || model.Enabled.ValueBool() || !model.PreRegistration.ValueBool() {
		t.Fatalf("boolean fields not mapped: %#v", model)
	}
}
