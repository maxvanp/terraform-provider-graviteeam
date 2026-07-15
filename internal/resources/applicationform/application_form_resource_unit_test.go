package applicationform

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplicationFormBuildUpdateBodyPreservesAssets(t *testing.T) {
	body := buildUpdateBody(
		ApplicationFormModel{
			Enabled: types.BoolValue(false),
			Content: types.StringValue("<html>updated app form</html>"),
		},
		map[string]interface{}{
			"assets": "asset-payload",
		},
	)

	if body["enabled"] != false {
		t.Fatalf("expected enabled to be managed, got %#v", body["enabled"])
	}
	if body["content"] != "<html>updated app form</html>" {
		t.Fatalf("expected content to be managed, got %#v", body["content"])
	}
	if body["assets"] != "asset-payload" {
		t.Fatalf("expected assets to be preserved, got %#v", body["assets"])
	}
}
