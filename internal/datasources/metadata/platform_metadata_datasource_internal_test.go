package metadata

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPlatformMetadataMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewPlatformMetadataDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_platform_metadata"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestPlatformMetadataSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewPlatformMetadataDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "kind", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "role_id", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "result_json", false, false, true)
}

func TestPlatformMetadataConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&PlatformMetadataDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestEnvironmentMetadataMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewEnvironmentMetadataDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_environment_metadata"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestEnvironmentMetadataSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewEnvironmentMetadataDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "kind", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "result_json", false, false, true)
}

func TestEnvironmentMetadataConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&EnvironmentMetadataDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

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

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
