package domainmetadata

import "testing"

func TestFormatJSONRawString(t *testing.T) {
	got, err := formatJSON([]byte("ABC123"))
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if got != `"ABC123"` {
		t.Fatalf("formatJSON() = %q, want %q", got, `"ABC123"`)
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
