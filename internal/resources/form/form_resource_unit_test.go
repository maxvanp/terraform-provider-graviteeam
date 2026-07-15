package form

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFormBuildUpdateBodyPreservesAssets(t *testing.T) {
	body := buildUpdateBody(
		FormModel{
			Enabled: types.BoolValue(true),
			Content: types.StringValue("<html>updated</html>"),
		},
		map[string]interface{}{
			"assets": "asset-payload",
		},
	)

	if body["enabled"] != true {
		t.Fatalf("expected enabled to be managed, got %#v", body["enabled"])
	}
	if body["content"] != "<html>updated</html>" {
		t.Fatalf("expected content to be managed, got %#v", body["content"])
	}
	if body["assets"] != "asset-payload" {
		t.Fatalf("expected assets to be preserved, got %#v", body["assets"])
	}
}
