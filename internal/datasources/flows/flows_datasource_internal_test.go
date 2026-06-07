package flows

import "testing"

func TestFormatFlowsProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatFlows([]interface{}{
		map[string]interface{}{"id": "login", "enabled": true},
	})
	if err != nil {
		t.Fatalf("format flows: %v", err)
	}
	want := "[\n  {\n    \"enabled\": true,\n    \"id\": \"login\"\n  }\n]"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFormatFlowsReturnsMarshalError(t *testing.T) {
	t.Parallel()

	_, err := formatFlows([]interface{}{func() {}})
	if err == nil {
		t.Fatalf("expected marshal error")
	}
}
