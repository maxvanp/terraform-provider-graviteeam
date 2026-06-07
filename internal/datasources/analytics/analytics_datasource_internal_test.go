package analytics

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildAnalyticsParamsUsesExplicitValues(t *testing.T) {
	t.Parallel()

	model := AnalyticsModel{
		Type:     types.StringValue("GROUP_BY"),
		Field:    types.StringValue("type"),
		From:     types.Int64Value(1000),
		To:       types.Int64Value(2000),
		Interval: types.Int64Value(300),
		Size:     types.Int64Value(5),
	}

	got := buildAnalyticsParams(model, 9999)
	want := map[string]string{
		"type":     "GROUP_BY",
		"field":    "type",
		"from":     "1000",
		"to":       "2000",
		"interval": "300",
		"size":     "5",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("params = %#v, want %#v", got, want)
	}
}

func TestBuildAnalyticsParamsUsesDefaultTimeWindowAndOmitsUnknownOptionals(t *testing.T) {
	t.Parallel()

	model := AnalyticsModel{
		Type:     types.StringValue("COUNT"),
		Field:    types.StringUnknown(),
		From:     types.Int64Null(),
		To:       types.Int64Unknown(),
		Interval: types.Int64Null(),
		Size:     types.Int64Unknown(),
	}

	got := buildAnalyticsParams(model, 172800000)
	want := map[string]string{
		"type": "COUNT",
		"from": "86400000",
		"to":   "172800000",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("params = %#v, want %#v", got, want)
	}
}

func TestFormatAnalyticsResultProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatAnalyticsResult(map[string]interface{}{
		"count": float64(2),
		"type":  "COUNT",
	})
	if err != nil {
		t.Fatalf("format analytics: %v", err)
	}
	want := "{\n  \"count\": 2,\n  \"type\": \"COUNT\"\n}"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFormatAnalyticsResultReturnsMarshalError(t *testing.T) {
	t.Parallel()

	_, err := formatAnalyticsResult(map[string]interface{}{"bad": func() {}})
	if err == nil {
		t.Fatalf("expected marshal error")
	}
}
