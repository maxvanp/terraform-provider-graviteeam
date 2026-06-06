package permissionsmetadata

import "testing"

func TestFormatJSONRawString(t *testing.T) {
	got, err := formatJSON([]byte("APPLICATION[READ]"))
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if got != `"APPLICATION[READ]"` {
		t.Fatalf("formatJSON() = %q, want %q", got, `"APPLICATION[READ]"`)
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
