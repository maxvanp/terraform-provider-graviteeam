package permissionsmetadata

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPermissionsMetadataPathsBuildRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		kind     string
		config   PermissionsMetadataModel
		wantPath string
	}{
		{
			name: "application member permissions",
			kind: "application_member_permissions",
			config: PermissionsMetadataModel{
				DomainID:      types.StringValue("domain-1"),
				ApplicationID: types.StringValue("app-1"),
			},
			wantPath: "/domains/domain-1/applications/app-1/members/permissions",
		},
		{
			name: "domain member permissions",
			kind: "domain_member_permissions",
			config: PermissionsMetadataModel{
				DomainID: types.StringValue("domain-1"),
			},
			wantPath: "/domains/domain-1/members/permissions",
		},
		{
			name:     "environment member permissions",
			kind:     "environment_member_permissions",
			config:   PermissionsMetadataModel{},
			wantPath: "/members/permissions",
		},
		{
			name: "protected resource member permissions",
			kind: "protected_resource_member_permissions",
			config: PermissionsMetadataModel{
				DomainID:            types.StringValue("domain-1"),
				ProtectedResourceID: types.StringValue("resource-1"),
			},
			wantPath: "/domains/domain-1/protected-resources/resource-1/members/permissions",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := permissionsMetadataPaths[tt.kind](tt.config)
			if err != nil {
				t.Fatalf("permissionsMetadataPaths[%q] returned error: %v", tt.kind, err)
			}
			if got != tt.wantPath {
				t.Fatalf("path = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestPermissionsMetadataPathsRejectMissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		kind    string
		config  PermissionsMetadataModel
		wantErr string
	}{
		{
			name: "application permissions requires domain",
			kind: "application_member_permissions",
			config: PermissionsMetadataModel{
				ApplicationID: types.StringValue("app-1"),
			},
			wantErr: "domain_id is required for application_member_permissions",
		},
		{
			name: "application permissions requires application",
			kind: "application_member_permissions",
			config: PermissionsMetadataModel{
				DomainID: types.StringValue("domain-1"),
			},
			wantErr: "application_id is required for application_member_permissions",
		},
		{
			name:    "domain permissions requires domain",
			kind:    "domain_member_permissions",
			config:  PermissionsMetadataModel{},
			wantErr: "domain_id is required for domain_member_permissions",
		},
		{
			name: "protected resource permissions requires protected resource",
			kind: "protected_resource_member_permissions",
			config: PermissionsMetadataModel{
				DomainID: types.StringValue("domain-1"),
			},
			wantErr: "protected_resource_id is required for protected_resource_member_permissions",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := permissionsMetadataPaths[tt.kind](tt.config)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestPermissionsMetadataKindListIsSorted(t *testing.T) {
	t.Parallel()

	got := permissionsMetadataKindList()
	want := "application_member_permissions, domain_member_permissions, environment_member_permissions, protected_resource_member_permissions"
	if got != want {
		t.Fatalf("kind list = %q, want %q", got, want)
	}
}

func TestFormatJSONRawString(t *testing.T) {
	t.Parallel()

	got, err := formatJSON([]byte("APPLICATION[READ]"))
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if got != `"APPLICATION[READ]"` {
		t.Fatalf("formatJSON() = %q, want %q", got, `"APPLICATION[READ]"`)
	}
}

func TestFormatJSONEmptyBody(t *testing.T) {
	t.Parallel()

	got, err := formatJSON(nil)
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if got != "null" {
		t.Fatalf("formatJSON() = %q, want null", got)
	}
}

func TestFormatJSONPrettyPrintsObjects(t *testing.T) {
	t.Parallel()

	got, err := formatJSON([]byte(`{"permissions":["READ","UPDATE"]}`))
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if !strings.Contains(got, "\n") || !strings.Contains(got, `"permissions": [`) || !strings.Contains(got, `"UPDATE"`) {
		t.Fatalf("formatJSON() = %q, want pretty JSON object", got)
	}
}
