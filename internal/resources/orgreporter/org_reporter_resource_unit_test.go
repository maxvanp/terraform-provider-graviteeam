package orgreporter

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOrgReporterBuildBodyUsesPlannedInherited(t *testing.T) {
	body := buildBody(
		OrgReporterModel{
			Name:          types.StringValue("reporter"),
			Type:          types.StringValue("reporter-am-file"),
			Configuration: types.StringValue(`{"filename":"audit.log"}`),
			Enabled:       types.BoolValue(false),
			Inherited:     types.BoolValue(false),
		},
		map[string]interface{}{"inherited": true},
	)

	if body["inherited"] != false {
		t.Fatalf("expected planned inherited value, got %#v", body["inherited"])
	}
}

func TestOrgReporterBuildBodyPreservesUnknownInherited(t *testing.T) {
	body := buildBody(
		OrgReporterModel{
			Name:          types.StringValue("reporter"),
			Type:          types.StringValue("reporter-am-file"),
			Configuration: types.StringValue(`{"filename":"audit.log"}`),
			Enabled:       types.BoolValue(false),
			Inherited:     types.BoolUnknown(),
		},
		map[string]interface{}{"inherited": true},
	)

	if body["inherited"] != true {
		t.Fatalf("expected inherited to be preserved when unknown, got %#v", body["inherited"])
	}
}

func TestOrgReporterBuildBodyPreservesAttributeMappings(t *testing.T) {
	body := buildBody(
		OrgReporterModel{
			Name:          types.StringValue("reporter"),
			Type:          types.StringValue("reporter-am-file"),
			Configuration: types.StringValue(`{}`),
			Enabled:       types.BoolValue(true),
			Inherited:     types.BoolValue(false),
		},
		map[string]interface{}{
			"attributeMappings":          []interface{}{map[string]interface{}{"name": "userId"}},
			"attributeMappingEventTypes": []interface{}{"USER_LOGIN"},
		},
	)

	if body["attributeMappings"] == nil || body["attributeMappingEventTypes"] == nil {
		t.Fatalf("expected attribute mappings to be preserved, got %#v", body)
	}
}
