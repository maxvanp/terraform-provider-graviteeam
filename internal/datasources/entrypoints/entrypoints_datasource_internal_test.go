package entrypoints

import "testing"

func TestFormatEntrypointsProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatEntrypoints([]byte(`[{"id":"default","url":"https://example.test"}]`))
	if err != nil {
		t.Fatalf("format entrypoints: %v", err)
	}
	want := "[\n  {\n    \"id\": \"default\",\n    \"url\": \"https://example.test\"\n  }\n]"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFormatEntrypointsRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := formatEntrypoints([]byte(`{`))
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
