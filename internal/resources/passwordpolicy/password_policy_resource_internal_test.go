package passwordpolicy

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		id                   string
		wantDomainID         string
		wantPasswordPolicyID string
		wantOK               bool
	}{
		{
			name:                 "valid",
			id:                   "domain-1/policy-1",
			wantDomainID:         "domain-1",
			wantPasswordPolicyID: "policy-1",
			wantOK:               true,
		},
		{
			name:                 "preserves existing splitN behavior",
			id:                   "domain-1/policy-1/extra",
			wantDomainID:         "domain-1",
			wantPasswordPolicyID: "policy-1/extra",
			wantOK:               true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotPasswordPolicyID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotPasswordPolicyID != tt.wantPasswordPolicyID {
				t.Fatalf("password policy ID = %q, want %q", gotPasswordPolicyID, tt.wantPasswordPolicyID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	resource := &PasswordPolicyResource{}
	plan := PasswordPolicyModel{
		Name:                             types.StringValue("Strict Policy"),
		MinLength:                        types.Int64Value(12),
		MaxLength:                        types.Int64Value(64),
		MaxConsecutiveLetters:            types.Int64Value(3),
		ExpiryDuration:                   types.Int64Value(3600),
		OldPasswords:                     types.Int64Value(5),
		IncludeNumbers:                   types.BoolValue(true),
		IncludeSpecialCharacters:         types.BoolValue(false),
		LettersInMixedCase:               types.BoolValue(true),
		ExcludePasswordsInDictionary:     types.BoolValue(false),
		ExcludeUserProfileInfoInPassword: types.BoolValue(true),
		PasswordHistoryEnabled:           types.BoolValue(true),
		DefaultPolicy:                    types.BoolValue(false),
	}

	got := resource.buildBody(plan, true)
	want := map[string]interface{}{
		"name":                             "Strict Policy",
		"minLength":                        int64(12),
		"maxLength":                        int64(64),
		"maxConsecutiveLetters":            int64(3),
		"expiryDuration":                   int64(3600),
		"oldPasswords":                     int64(5),
		"includeNumbers":                   true,
		"includeSpecialCharacters":         false,
		"lettersInMixedCase":               true,
		"excludePasswordsInDictionary":     false,
		"excludeUserProfileInfoInPassword": true,
		"passwordHistoryEnabled":           true,
		"defaultPolicy":                    false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyOmitsDefaultPolicyWhenCreateUsesDedicatedEndpoint(t *testing.T) {
	t.Parallel()

	resource := &PasswordPolicyResource{}
	plan := PasswordPolicyModel{
		Name:          types.StringValue("Default Policy"),
		DefaultPolicy: types.BoolValue(true),
		MinLength:     types.Int64Null(),
	}

	got := resource.buildBody(plan, false)
	if _, ok := got["defaultPolicy"]; ok {
		t.Fatalf("defaultPolicy should be omitted from create body: %#v", got)
	}
	if _, ok := got["minLength"]; ok {
		t.Fatalf("null minLength should be omitted from body: %#v", got)
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	resource := &PasswordPolicyResource{}
	model := PasswordPolicyModel{
		DomainID: types.StringValue("domain-1"),
		Name:     types.StringValue("old-name"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":                               "policy-1",
		"name":                             "Strict Policy",
		"minLength":                        float64(12),
		"maxLength":                        int64(64),
		"maxConsecutiveLetters":            float64(3),
		"expiryDuration":                   int64(3600),
		"oldPasswords":                     float64(5),
		"includeNumbers":                   true,
		"includeSpecialCharacters":         false,
		"lettersInMixedCase":               true,
		"excludePasswordsInDictionary":     false,
		"excludeUserProfileInfoInPassword": true,
		"passwordHistoryEnabled":           true,
		"defaultPolicy":                    false,
		"ignoredUnsupportedNumericRepresentation": int(2),
	})

	assertString(t, model.ID, "id", "policy-1")
	assertString(t, model.Name, "name", "Strict Policy")
	assertInt64(t, model.MinLength, "minLength", 12)
	assertInt64(t, model.MaxLength, "maxLength", 64)
	assertInt64(t, model.MaxConsecutiveLetters, "maxConsecutiveLetters", 3)
	assertInt64(t, model.ExpiryDuration, "expiryDuration", 3600)
	assertInt64(t, model.OldPasswords, "oldPasswords", 5)
	assertBool(t, model.IncludeNumbers, "includeNumbers", true)
	assertBool(t, model.IncludeSpecialCharacters, "includeSpecialCharacters", false)
	assertBool(t, model.LettersInMixedCase, "lettersInMixedCase", true)
	assertBool(t, model.ExcludePasswordsInDictionary, "excludePasswordsInDictionary", false)
	assertBool(t, model.ExcludeUserProfileInfoInPassword, "excludeUserProfileInfoInPassword", true)
	assertBool(t, model.PasswordHistoryEnabled, "passwordHistoryEnabled", true)
	assertBool(t, model.DefaultPolicy, "defaultPolicy", false)
}

func TestReadMissingValuesAsNull(t *testing.T) {
	t.Parallel()

	resource := &PasswordPolicyResource{}
	model := PasswordPolicyModel{}

	resource.readIntoModel(&model, map[string]interface{}{"name": "Sparse Policy"})

	if !model.MinLength.IsNull() {
		t.Fatalf("missing minLength should be null")
	}
	if !model.IncludeNumbers.IsNull() {
		t.Fatalf("missing includeNumbers should be null")
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
