package formpreview

import "testing"

func TestFormatJSONHandlesEmptyJSONAndText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty": {nil, "null"},
		"text":  {[]byte("<html>preview</html>"), `"\u003chtml\u003epreview\u003c/html\u003e"`},
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

func TestFormatJSONProducesStableIndentedPreviewJSON(t *testing.T) {
	t.Parallel()

	got, err := formatJSON([]byte(`{"content":"ok","type":"FORM"}`))
	if err != nil {
		t.Fatalf("format json: %v", err)
	}
	want := "{\n  \"content\": \"ok\",\n  \"type\": \"FORM\"\n}"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}
