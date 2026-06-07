package plugins

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatJSONResultHandlesEmptyJSONAndText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty": {nil, "null"},
		"text":  {[]byte("plain text"), `"plain text"`},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := formatJSONResult(test.raw)
			if err != nil {
				t.Fatalf("format result: %v", err)
			}
			if got != test.want {
				t.Fatalf("result = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatJSONResultProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatJSONResult([]byte(`{"id":"ldap","enabled":true}`))
	if err != nil {
		t.Fatalf("format result: %v", err)
	}
	want := "{\n  \"enabled\": true,\n  \"id\": \"ldap\"\n}"
	if got != want {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPluginCategoryListContainsEveryAllowedCategory(t *testing.T) {
	t.Parallel()

	got := pluginCategoryList()
	for category := range allowedPluginCategories {
		if !strings.Contains(got, category) {
			t.Fatalf("category list %q does not contain %q", got, category)
		}
	}
}

func TestFormatJSONResultEscapesInvalidJSONText(t *testing.T) {
	t.Parallel()

	got, err := formatJSONResult([]byte("line\nbreak"))
	if err != nil {
		t.Fatalf("format result: %v", err)
	}
	var decoded string
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("formatted text is not a JSON string: %v", err)
	}
	if decoded != "line\nbreak" {
		t.Fatalf("decoded = %q, want original text", decoded)
	}
}
