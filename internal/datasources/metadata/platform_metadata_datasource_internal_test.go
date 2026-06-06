package metadata

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPlatformMetadataRolePath(t *testing.T) {
	config := PlatformMetadataModel{
		RoleID: types.StringValue("role/id with spaces"),
	}

	path, err := platformMetadataPaths["role"](config)
	if err != nil {
		t.Fatalf("role path returned error: %v", err)
	}

	want := "platform/roles/role%2Fid%20with%20spaces"
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestPlatformMetadataRoleRequiresRoleID(t *testing.T) {
	_, err := platformMetadataPaths["role"](PlatformMetadataModel{})
	if err == nil {
		t.Fatal("expected error for missing role_id")
	}
}
