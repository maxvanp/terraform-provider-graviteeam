package usercertificatecredential

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

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewUserCertificateCredentialResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_user_certificate_credential"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewUserCertificateCredentialResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "user_id", "certificate_pem"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{
		"id",
		"certificate_thumbprint",
		"certificate_subject_dn",
		"certificate_serial_number",
		"certificate_issuer_dn",
		"certificate_expires_at",
		"username",
	} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&UserCertificateCredentialResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := UserCertificateCredentialModel{
		CertificatePEM: types.StringValue("-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n"),
	}

	got := buildCreateBody(plan)
	want := map[string]interface{}{
		"certificatePem": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelMapsCertificateCredential(t *testing.T) {
	t.Parallel()

	model := UserCertificateCredentialModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":                      "credential-id",
		"certificatePem":          "pem",
		"certificateThumbprint":   "thumbprint",
		"certificateSubjectDN":    "CN=subject",
		"certificateSerialNumber": "serial",
		"certificateIssuerDN":     "CN=issuer",
		"certificateExpiresAt":    "2026-06-07T12:00:00Z",
		"username":                "alice",
	})

	assertStringAttr(t, model.ID, "id", "credential-id")
	assertStringAttr(t, model.CertificatePEM, "certificate pem", "pem")
	assertStringAttr(t, model.CertificateThumbprint, "certificate thumbprint", "thumbprint")
	assertStringAttr(t, model.CertificateSubjectDN, "certificate subject dn", "CN=subject")
	assertStringAttr(t, model.CertificateSerialNumber, "certificate serial number", "serial")
	assertStringAttr(t, model.CertificateIssuerDN, "certificate issuer dn", "CN=issuer")
	assertStringAttr(t, model.CertificateExpiresAt, "certificate expires at", "2026-06-07T12:00:00Z")
	assertStringAttr(t, model.Username, "username", "alice")
}

func TestReadIntoModelNullsMissingComputedFields(t *testing.T) {
	t.Parallel()

	model := UserCertificateCredentialModel{
		CertificateThumbprint:   types.StringValue("thumbprint"),
		CertificateSubjectDN:    types.StringValue("CN=subject"),
		CertificateSerialNumber: types.StringValue("serial"),
		CertificateIssuerDN:     types.StringValue("CN=issuer"),
		CertificateExpiresAt:    types.StringValue("2026-06-07T12:00:00Z"),
		Username:                types.StringValue("alice"),
	}

	readIntoModel(&model, map[string]interface{}{})

	assertNullString(t, model.CertificateThumbprint, "certificate thumbprint")
	assertNullString(t, model.CertificateSubjectDN, "certificate subject dn")
	assertNullString(t, model.CertificateSerialNumber, "certificate serial number")
	assertNullString(t, model.CertificateIssuerDN, "certificate issuer dn")
	assertNullString(t, model.CertificateExpiresAt, "certificate expires at")
	assertNullString(t, model.Username, "username")
}

func TestTimestampString(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input interface{}
		want  string
		null  bool
	}{
		"string": {
			input: "2026-06-07T12:00:00Z",
			want:  "2026-06-07T12:00:00Z",
		},
		"float": {
			input: float64(1780833600000),
			want:  "1780833600000",
		},
		"nil": {
			input: nil,
			null:  true,
		},
		"malformed": {
			input: map[string]interface{}{},
			null:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := timestampString(tc.input)
			if tc.null {
				assertNullString(t, got, name)
				return
			}
			assertStringAttr(t, got, name, tc.want)
		})
	}
}

func TestUserCertificateCredentialCRUDAndUnsupportedUpdate(t *testing.T) {
	const certificatePEM = "-----BEGIN CERTIFICATE-----\nMIID\n-----END CERTIFICATE-----\n"

	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/cert-credentials", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(certificateCredentialResponse("credential-123", certificatePEM))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/cert-credentials/credential-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(certificateCredentialResponse("credential-123", certificatePEM))
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserCertificateCredentialResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := userCertificateCredentialPlan(t, schemaResp.Schema, UserCertificateCredentialModel{
		DomainID:       types.StringValue("domain-123"),
		UserID:         types.StringValue("user-123"),
		CertificatePEM: types.StringValue(certificatePEM),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState UserCertificateCredentialModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	assertStringAttr(t, readState.ID, "id", "credential-123")
	assertStringAttr(t, readState.CertificateThumbprint, "certificate thumbprint", "thumbprint")
	assertStringAttr(t, readState.Username, "username", "alice")

	updateResp := &resource.UpdateResponse{}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected unsupported update diagnostic")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: readResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if !reflect.DeepEqual(methods, []string{"create", "read", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	wantBodies := []map[string]interface{}{{"certificatePem": certificatePEM}}
	if !reflect.DeepEqual(bodies, wantBodies) {
		t.Fatalf("bodies = %#v, want %#v", bodies, wantBodies)
	}
}

func TestUserCertificateCredentialReadRemovesMissingResource(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/cert-credentials/missing", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserCertificateCredentialResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &UserCertificateCredentialModel{
		ID:             types.StringValue("missing"),
		DomainID:       types.StringValue("domain-123"),
		UserID:         types.StringValue("user-123"),
		CertificatePEM: types.StringValue("pem"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing resource to remove state, got %#v", readResp.State.Raw)
	}
}

func TestUserCertificateCredentialReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/cert-credentials", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/cert-credentials/credential-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			http.Error(w, "read failed", http.StatusInternalServerError)
		case http.MethodDelete:
			http.Error(w, "delete failed", http.StatusInternalServerError)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserCertificateCredentialResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := userCertificateCredentialPlan(t, schemaResp.Schema, UserCertificateCredentialModel{
		DomainID:       types.StringValue("domain-123"),
		UserID:         types.StringValue("user-123"),
		CertificatePEM: types.StringValue("pem"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := userCertificateCredentialState(t, schemaResp.Schema, UserCertificateCredentialModel{
		ID:             types.StringValue("credential-123"),
		DomainID:       types.StringValue("domain-123"),
		UserID:         types.StringValue("user-123"),
		CertificatePEM: types.StringValue("pem"),
	})
	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestUserCertificateCredentialDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/cert-credentials/credential-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("item method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserCertificateCredentialResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := userCertificateCredentialState(t, schemaResp.Schema, UserCertificateCredentialModel{
		ID:             types.StringValue("credential-123"),
		DomainID:       types.StringValue("domain-123"),
		UserID:         types.StringValue("user-123"),
		CertificatePEM: types.StringValue("pem"),
	})

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestUserCertificateCredentialImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&UserCertificateCredentialResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/user-123",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func certificateCredentialResponse(id, certificatePEM string) map[string]interface{} {
	return map[string]interface{}{
		"id":                      id,
		"certificatePem":          certificatePEM,
		"certificateThumbprint":   "thumbprint",
		"certificateSubjectDN":    "CN=subject",
		"certificateSerialNumber": "serial",
		"certificateIssuerDN":     "CN=issuer",
		"certificateExpiresAt":    "2026-06-07T12:00:00Z",
		"username":                "alice",
	}
}

func userCertificateCredentialPlan(t *testing.T, schema resourceschema.Schema, model UserCertificateCredentialModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func userCertificateCredentialState(t *testing.T, schema resourceschema.Schema, model UserCertificateCredentialModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func assertStringAttr(t *testing.T, got types.String, name, want string) {
	t.Helper()

	if got.IsNull() || got.IsUnknown() {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
	if got.ValueString() != want {
		t.Fatalf("%s = %q, want %q", name, got.ValueString(), want)
	}
}

func assertNullString(t *testing.T, got types.String, name string) {
	t.Helper()

	if !got.IsNull() {
		t.Fatalf("%s should be null, got %q", name, got.ValueString())
	}
}
