package identityproviderpasswordpolicy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIdentityProviderPasswordPolicyMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewIdentityProviderPasswordPolicyResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_identity_provider_password_policy" {
		t.Fatalf("type name = %q, want graviteeam_identity_provider_password_policy", resp.TypeName)
	}
}

func TestIdentityProviderPasswordPolicySchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewIdentityProviderPasswordPolicyResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "identity_provider_id", "password_policy_id"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; !attr.IsComputed() {
		t.Fatal("id should be computed")
	}
}

func TestIdentityProviderPasswordPolicyConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &IdentityProviderPasswordPolicyResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestParseAssignmentImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		id             string
		wantDomainID   string
		wantProviderID string
		wantOK         bool
	}{
		{
			name:           "valid",
			id:             "domain-1/idp-1",
			wantDomainID:   "domain-1",
			wantProviderID: "idp-1",
			wantOK:         true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
		{
			name:   "too many segments",
			id:     "domain-1/idp-1/extra",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotProviderID, gotOK := parseAssignmentImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotProviderID != tt.wantProviderID {
				t.Fatalf("identity provider ID = %q, want %q", gotProviderID, tt.wantProviderID)
			}
		})
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	model := IdentityProviderPasswordPolicyModel{
		DomainID:           types.StringValue("domain-1"),
		IdentityProviderID: types.StringValue("idp-1"),
		PasswordPolicyID:   types.StringValue("old-policy"),
	}

	readIntoModel(&model, "policy-2")

	if got, want := model.ID.ValueString(), "domain-1/idp-1"; got != want {
		t.Fatalf("ID = %q, want %q", got, want)
	}
	if got, want := model.PasswordPolicyID.ValueString(), "policy-2"; got != want {
		t.Fatalf("password policy ID = %q, want %q", got, want)
	}
}
