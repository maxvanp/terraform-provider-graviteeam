package usercollections

import "testing"

func TestFormatCollectionItemsProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatCollectionItems([]byte(`[{"id":"item-1","type":"credential"}]`))
	if err != nil {
		t.Fatalf("format collection items: %v", err)
	}
	want := "[\n  {\n    \"id\": \"item-1\",\n    \"type\": \"credential\"\n  }\n]"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFormatCollectionItemsRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := formatCollectionItems([]byte(`{`))
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
