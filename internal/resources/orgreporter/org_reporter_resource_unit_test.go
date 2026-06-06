package orgreporter

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOrgReporterBuildBodyPreservesInherited(t *testing.T) {
	body := buildBody(
		OrgReporterModel{
			Name:          types.StringValue("reporter"),
			Type:          types.StringValue("reporter-am-file"),
			Configuration: types.StringValue(`{"filename":"audit.log"}`),
			Enabled:       types.BoolValue(false),
		},
		map[string]interface{}{"inherited": true},
	)

	if body["inherited"] != true {
		t.Fatalf("expected inherited to be preserved, got %#v", body["inherited"])
	}
}
