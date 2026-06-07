package passwordpolicyevaluation

import "testing"

func TestFormatJSONHandlesEmptyJSONAndText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty": {nil, "null"},
		"text":  {[]byte("not json"), `"not json"`},
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

func TestFormatJSONProducesStableIndentedEvaluationJSON(t *testing.T) {
	t.Parallel()

	got, err := formatJSON([]byte(`{"valid":false,"errors":["too_short"]}`))
	if err != nil {
		t.Fatalf("format json: %v", err)
	}
	want := "{\n  \"errors\": [\n    \"too_short\"\n  ],\n  \"valid\": false\n}"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}
