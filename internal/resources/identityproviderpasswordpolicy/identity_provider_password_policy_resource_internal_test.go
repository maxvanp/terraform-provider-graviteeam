package identityproviderpasswordpolicy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
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

func TestAssignmentIDUsesDomainAndIdentityProvider(t *testing.T) {
	t.Parallel()

	got := assignmentID("domain-1", "idp-1")

	if got.ValueString() != "domain-1/idp-1" {
		t.Fatalf("assignment ID = %q, want domain-1/idp-1", got.ValueString())
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

func TestIdentityProviderPasswordPolicyCRUDAssignsAndClearsRelationship(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})
	var putBodies []map[string]interface{}
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities/idp-123/password-policy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT password-policy relationship, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode password-policy relationship body: %v", err)
		}
		putBodies = append(putBodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"updated": true})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities/idp-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET identity provider, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "idp-123", "passwordPolicy": "policy-read"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &IdentityProviderPasswordPolicyResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := identityProviderPasswordPolicyPlan(t, schemaResp.Schema, IdentityProviderPasswordPolicyModel{
		DomainID:           types.StringValue("domain-123"),
		IdentityProviderID: types.StringValue("idp-123"),
		PasswordPolicyID:   types.StringValue("policy-create"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var created IdentityProviderPasswordPolicyModel
	if diags := createResp.State.Get(context.Background(), &created); diags.HasError() {
		t.Fatalf("get created state: %#v", diags)
	}
	if created.ID.ValueString() != "domain-123/idp-123" {
		t.Fatalf("created id = %q", created.ID.ValueString())
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var read IdentityProviderPasswordPolicyModel
	if diags := readResp.State.Get(context.Background(), &read); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if read.PasswordPolicyID.ValueString() != "policy-read" {
		t.Fatalf("read password policy = %q", read.PasswordPolicyID.ValueString())
	}

	updatePlan := identityProviderPasswordPolicyPlan(t, schemaResp.Schema, IdentityProviderPasswordPolicyModel{
		DomainID:           types.StringValue("domain-123"),
		IdentityProviderID: types.StringValue("idp-123"),
		PasswordPolicyID:   types.StringValue("policy-update"),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: updatePlan, State: readResp.State}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}

	deleteResp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
	if len(putBodies) != 3 {
		t.Fatalf("PUT bodies = %#v, want create, update, delete", putBodies)
	}
	if !reflect.DeepEqual(putBodies[0], map[string]interface{}{"passwordPolicy": "policy-create"}) {
		t.Fatalf("create body = %#v", putBodies[0])
	}
	if !reflect.DeepEqual(putBodies[1], map[string]interface{}{"passwordPolicy": "policy-update"}) {
		t.Fatalf("update body = %#v", putBodies[1])
	}
	if value, ok := putBodies[2]["passwordPolicy"]; !ok || value != nil {
		t.Fatalf("delete body = %#v", putBodies[2])
	}
}

func identityProviderPasswordPolicyPlan(t *testing.T, schema resourceschema.Schema, model IdentityProviderPasswordPolicyModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
