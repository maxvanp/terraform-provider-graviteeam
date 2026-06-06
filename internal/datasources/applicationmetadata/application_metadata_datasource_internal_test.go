package applicationmetadata

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplicationMetadataResourcePoliciesPath(t *testing.T) {
	config := ApplicationMetadataModel{
		ResourceID: types.StringValue("resource/id with spaces"),
	}

	path, err := applicationMetadataPaths["resource_policies"](config)
	if err != nil {
		t.Fatalf("resource_policies path returned error: %v", err)
	}

	want := "/resources/resource%2Fid%20with%20spaces/policies"
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestApplicationMetadataResourcePoliciesRequiresResourceID(t *testing.T) {
	_, err := applicationMetadataPaths["resource_policies"](ApplicationMetadataModel{})
	if err == nil {
		t.Fatal("expected error for missing resource_id")
	}
}
