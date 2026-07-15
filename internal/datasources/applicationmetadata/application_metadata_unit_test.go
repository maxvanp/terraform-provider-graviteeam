package applicationmetadata

import (
	"net/url"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplicationMetadataPathsBuildRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		kind     string
		config   ApplicationMetadataModel
		wantPath string
	}{
		{
			name: "analytics uses provided query values",
			kind: "analytics",
			config: ApplicationMetadataModel{
				Type:     types.StringValue("DATE_HISTO"),
				Field:    types.StringValue("status"),
				From:     types.Int64Value(1000),
				To:       types.Int64Value(2000),
				Interval: types.Int64Value(100),
				Size:     types.Int64Value(20),
			},
			wantPath: "/analytics?field=status&from=1000&interval=100&size=20&to=2000&type=DATE_HISTO",
		},
		{
			name:     "resources uses default page size",
			kind:     "resources",
			config:   ApplicationMetadataModel{},
			wantPath: "/resources?page=0&size=50",
		},
		{
			name: "resource policies escapes resource id",
			kind: "resource_policies",
			config: ApplicationMetadataModel{
				ResourceID: types.StringValue("resource/id with spaces"),
			},
			wantPath: "/resources/resource%2Fid%20with%20spaces/policies",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := applicationMetadataPaths[tt.kind](tt.config)
			if err != nil {
				t.Fatalf("applicationMetadataPaths[%q] returned error: %v", tt.kind, err)
			}
			if got != tt.wantPath {
				t.Fatalf("path = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestApplicationMetadataResourcePoliciesRequiresResourceIDUnit(t *testing.T) {
	t.Parallel()

	_, err := applicationMetadataPaths["resource_policies"](ApplicationMetadataModel{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Error() != "resource_id is required for resource_policies" {
		t.Fatalf("error = %q", err)
	}
}

func TestApplicationMetadataAnalyticsQueryDefaults(t *testing.T) {
	t.Parallel()

	query := strings.TrimPrefix(analyticsQuery(ApplicationMetadataModel{}), "?")
	values, err := url.ParseQuery(query)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}

	if got, want := values.Get("type"), "GROUP_BY"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := values.Get("field"), "application"; got != want {
		t.Fatalf("field = %q, want %q", got, want)
	}
	if values.Get("from") == "" || values.Get("to") == "" {
		t.Fatalf("from/to should be set in query: %s", query)
	}
	if values.Get("interval") != "" || values.Get("size") != "" {
		t.Fatalf("interval/size should be omitted by default: %s", query)
	}
}

func TestApplicationMetadataKindListIsSorted(t *testing.T) {
	t.Parallel()

	got := applicationMetadataKindList()
	want := "analytics, resource_policies, resources"
	if got != want {
		t.Fatalf("kind list = %q, want %q", got, want)
	}
}

func TestFormatJSONRawString(t *testing.T) {
	t.Parallel()

	got := formatJSON([]byte("raw-response"))
	if got != `"raw-response"` {
		t.Fatalf("formatJSON() = %q, want %q", got, `"raw-response"`)
	}
}

func TestFormatJSONEmptyBody(t *testing.T) {
	t.Parallel()

	got := formatJSON(nil)
	if got != "null" {
		t.Fatalf("formatJSON() = %q, want null", got)
	}
}

func TestFormatJSONPrettyPrintsObjects(t *testing.T) {
	t.Parallel()

	got := formatJSON([]byte(`{"resources":[{"id":"resource-1"}]}`))
	if !strings.Contains(got, "\n") || !strings.Contains(got, `"resources": [`) || !strings.Contains(got, `"resource-1"`) {
		t.Fatalf("formatJSON() = %q, want pretty JSON object", got)
	}
}
