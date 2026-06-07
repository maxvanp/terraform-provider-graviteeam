package adminmetadata

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAdminMetadataPathsBuildRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		kind      string
		config    AdminMetadataModel
		wantScope string
		wantPath  string
	}{
		{
			name: "domain by hrid escapes path segment",
			kind: "domain_by_hrid",
			config: AdminMetadataModel{
				HRID: types.StringValue("domain/hr id"),
			},
			wantScope: "environment",
			wantPath:  "/domains/_hrid/domain%2Fhr%20id",
		},
		{
			name:      "organization audits uses default page size",
			kind:      "organization_audits",
			config:    AdminMetadataModel{},
			wantScope: "organization",
			wantPath:  "/audits?page=0&size=10",
		},
		{
			name:      "organization environments",
			kind:      "organization_environments",
			config:    AdminMetadataModel{},
			wantScope: "organization",
			wantPath:  "/environments",
		},
		{
			name: "user audits escapes ids and uses configured page size",
			kind: "user_audits",
			config: AdminMetadataModel{
				DomainID: types.StringValue("domain/1"),
				UserID:   types.StringValue("user@example.com"),
				Size:     types.Int64Value(25),
			},
			wantScope: "environment",
			wantPath:  "/domains/domain%2F1/users/user@example.com/audits?page=0&size=25",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := adminMetadataPaths[tt.kind](tt.config)
			if err != nil {
				t.Fatalf("adminMetadataPaths[%q] returned error: %v", tt.kind, err)
			}
			if got.scope != tt.wantScope {
				t.Fatalf("scope = %q, want %q", got.scope, tt.wantScope)
			}
			if got.path != tt.wantPath {
				t.Fatalf("path = %q, want %q", got.path, tt.wantPath)
			}
		})
	}
}

func TestAdminMetadataPathsRejectMissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		kind    string
		config  AdminMetadataModel
		wantErr string
	}{
		{
			name:    "domain by hrid requires hrid",
			kind:    "domain_by_hrid",
			config:  AdminMetadataModel{},
			wantErr: "hrid is required for domain_by_hrid",
		},
		{
			name: "user audits requires domain",
			kind: "user_audits",
			config: AdminMetadataModel{
				UserID: types.StringValue("user-1"),
			},
			wantErr: "domain_id is required for user_audits",
		},
		{
			name: "user audits requires user",
			kind: "user_audits",
			config: AdminMetadataModel{
				DomainID: types.StringValue("domain-1"),
			},
			wantErr: "user_id is required for user_audits",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := adminMetadataPaths[tt.kind](tt.config)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestAdminMetadataKindListIsSorted(t *testing.T) {
	t.Parallel()

	got := adminMetadataKindList()
	want := "domain_by_hrid, organization_audits, organization_environments, user_audits"
	if got != want {
		t.Fatalf("kind list = %q, want %q", got, want)
	}
}

func TestFormatJSONRawString(t *testing.T) {
	t.Parallel()

	got, err := formatJSON([]byte("raw-response"))
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if got != `"raw-response"` {
		t.Fatalf("formatJSON() = %q, want %q", got, `"raw-response"`)
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

	got, err := formatJSON([]byte(`{"z":2,"a":{"b":1}}`))
	if err != nil {
		t.Fatalf("formatJSON returned error: %v", err)
	}
	if !strings.Contains(got, "\n") || !strings.Contains(got, `"a": {`) || !strings.Contains(got, `"z": 2`) {
		t.Fatalf("formatJSON() = %q, want pretty JSON object", got)
	}
}
