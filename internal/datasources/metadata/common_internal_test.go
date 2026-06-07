package metadata

import "testing"

func TestFormatJSONHandlesEmptyAndIndentedJSON(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty": {nil, "null"},
		"object": {[]byte(`{"id":"metadata","enabled":true}`),
			"{\n  \"enabled\": true,\n  \"id\": \"metadata\"\n}"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := formatJSON(test.raw)
			if err != nil {
				t.Fatalf("format json: %v", err)
			}
			if got != test.want {
				t.Fatalf("json = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatJSONRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := formatJSON([]byte(`{`))
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
