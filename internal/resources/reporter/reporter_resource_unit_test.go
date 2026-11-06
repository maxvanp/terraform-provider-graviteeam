package reporter

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestReporterBuildBodyUsesPlannedInherited(t *testing.T) {
	body := buildBody(
		ReporterModel{
			Name:          types.StringValue("reporter"),
			Type:          types.StringValue("reporter-am-file"),
			Configuration: types.StringValue(`{"filename":"audit.log"}`),
			Enabled:       types.BoolValue(true),
			Inherited:     types.BoolValue(false),
		},
		map[string]interface{}{"inherited": true},
	)

	if body["inherited"] != false {
		t.Fatalf("expected planned inherited value, got %#v", body["inherited"])
	}
}

func TestReporterBuildBodyPreservesUnknownInherited(t *testing.T) {
	body := buildBody(
		ReporterModel{
			Name:          types.StringValue("reporter"),
			Type:          types.StringValue("reporter-am-file"),
			Configuration: types.StringValue(`{"filename":"audit.log"}`),
			Enabled:       types.BoolValue(true),
			Inherited:     types.BoolUnknown(),
		},
		map[string]interface{}{"inherited": true},
	)

	if body["inherited"] != true {
		t.Fatalf("expected inherited to be preserved when unknown, got %#v", body["inherited"])
	}
}

func TestReporterBuildBodyPreservesAttributeMappings(t *testing.T) {
	mappings := []interface{}{map[string]interface{}{"name": "userId", "expression": "{#user.id}"}}
	eventTypes := []interface{}{"USER_LOGIN"}
	body := buildBody(
		ReporterModel{
			Name:          types.StringValue("reporter"),
			Type:          types.StringValue("reporter-am-file"),
			Configuration: types.StringValue(`{}`),
			Enabled:       types.BoolValue(true),
			Inherited:     types.BoolValue(false),
		},
		map[string]interface{}{
			"attributeMappings":          mappings,
			"attributeMappingEventTypes": eventTypes,
		},
	)

	if body["attributeMappings"] == nil || body["attributeMappingEventTypes"] == nil {
		t.Fatalf("expected attribute mappings to be preserved, got %#v", body)
	}
}
