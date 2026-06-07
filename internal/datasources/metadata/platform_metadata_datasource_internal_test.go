package metadata

import (
	"strings"
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

func TestMetadataKindListsAreSortedAndComplete(t *testing.T) {
	t.Parallel()

	got := platformMetadataKindList()
	want := "alert_service_status, audit_event_types, email_required, flow_schema, installation, license, role, spel_grammar"
	if got != want {
		t.Fatalf("platform kinds = %q, want %q", got, want)
	}

	envKinds := metadataKindList(environmentMetadataPaths)
	if envKinds != "data_planes, data_sources" {
		t.Fatalf("environment kinds = %q", envKinds)
	}
}

func TestEnvironmentMetadataPathsAreStable(t *testing.T) {
	t.Parallel()

	for kind, path := range map[string]string{
		"data_planes":  "data-planes",
		"data_sources": "data-sources",
	} {
		if environmentMetadataPaths[kind] != path {
			t.Fatalf("%s path = %q, want %q", kind, environmentMetadataPaths[kind], path)
		}
	}
}

func TestMetadataErrorMessage(t *testing.T) {
	t.Parallel()

	err := metadataError("missing value")
	if err.Error() != "missing value" || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %q", err.Error())
	}
}
