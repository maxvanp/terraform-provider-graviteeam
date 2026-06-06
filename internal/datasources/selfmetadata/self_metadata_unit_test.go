package selfmetadata

import "testing"

func TestFormatJSONRawString(t *testing.T) {
	got, err := formatJSON([]byte("plain"))
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if got != `"plain"` {
		t.Fatalf("formatJSON() = %q, want %q", got, `"plain"`)
	}
}

func TestFormatJSONEmptyBody(t *testing.T) {
	got, err := formatJSON(nil)
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if got != "null" {
		t.Fatalf("formatJSON() = %q, want null", got)
	}
}
