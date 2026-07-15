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
