package jsonformat

import "testing"

func TestString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty":        {want: "null"},
		"object":       {raw: []byte(`{"z":2,"a":{"b":1}}`), want: "{\n  \"a\": {\n    \"b\": 1\n  },\n  \"z\": 2\n}"},
		"scalar":       {raw: []byte(`true`), want: "true"},
		"invalid JSON": {raw: []byte("line\nbreak"), want: `"line\nbreak"`},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := String(test.raw); got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
		})
	}
}
