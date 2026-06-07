package audits

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAuditSizeUsesDefaultAndExplicitValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		model AuditsModel
		want  int
	}{
		"default null":    {AuditsModel{Size: types.Int64Null()}, 10},
		"default unknown": {AuditsModel{Size: types.Int64Unknown()}, 10},
		"explicit":        {AuditsModel{Size: types.Int64Value(5)}, 5},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := auditSize(test.model); got != test.want {
				t.Fatalf("size = %d, want %d", got, test.want)
			}
		})
	}
}

func TestExtractAuditEntriesReturnsDataSliceOnly(t *testing.T) {
	t.Parallel()

	entries := []interface{}{
		map[string]interface{}{"id": "audit-1"},
	}

	if got := extractAuditEntries(map[string]interface{}{"data": entries}); !reflect.DeepEqual(got, entries) {
		t.Fatalf("entries = %#v, want %#v", got, entries)
	}
	if got := extractAuditEntries(map[string]interface{}{"data": "not-a-list"}); got != nil {
		t.Fatalf("entries = %#v, want nil", got)
	}
	if got := extractAuditEntries(map[string]interface{}{}); got != nil {
		t.Fatalf("entries = %#v, want nil", got)
	}
}

func TestFormatAuditEntriesProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatAuditEntries(map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"id": "audit-1", "type": "USER_CREATED"},
		},
	})
	if err != nil {
		t.Fatalf("format audits: %v", err)
	}
	want := "[\n  {\n    \"id\": \"audit-1\",\n    \"type\": \"USER_CREATED\"\n  }\n]"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFormatAuditEntriesUsesNullWhenDataIsMissing(t *testing.T) {
	t.Parallel()

	got, err := formatAuditEntries(map[string]interface{}{})
	if err != nil {
		t.Fatalf("format audits: %v", err)
	}
	if got != "null" {
		t.Fatalf("json = %q, want null", got)
	}
}
