package applicationmetadata

import "testing"

func TestFormatJSONRawString(t *testing.T) {
	got, err := formatJSON([]byte("raw-response"))
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if got != `"raw-response"` {
		t.Fatalf("formatJSON() = %q, want %q", got, `"raw-response"`)
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
