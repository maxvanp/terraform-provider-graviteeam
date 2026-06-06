package reporter

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestReporterBuildBodyPreservesInherited(t *testing.T) {
	body := buildBody(
		ReporterModel{
			Name:          types.StringValue("reporter"),
			Type:          types.StringValue("reporter-am-file"),
			Configuration: types.StringValue(`{"filename":"audit.log"}`),
			Enabled:       types.BoolValue(true),
		},
		map[string]interface{}{"inherited": true},
	)

	if body["inherited"] != true {
		t.Fatalf("expected inherited to be preserved, got %#v", body["inherited"])
	}
}
