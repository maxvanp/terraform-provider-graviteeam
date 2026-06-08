package certificate

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
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestCertificateMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewCertificateResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_certificate" {
		t.Fatalf("type name = %q, want graviteeam_certificate", resp.TypeName)
	}
}

func TestCertificateSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewCertificateResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "type", "configuration"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["configuration"]; !attr.IsSensitive() {
		t.Fatal("configuration should be sensitive")
	}
	if attr := resp.Schema.Attributes["id"]; !attr.IsComputed() {
		t.Fatal("id should be computed")
	}
}

func TestCertificateConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &CertificateResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestCertificateConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&CertificateResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		id                string
		wantDomainID      string
		wantCertificateID string
		wantOK            bool
	}{
		{
			name:              "valid",
			id:                "domain-1/cert-1",
			wantDomainID:      "domain-1",
			wantCertificateID: "cert-1",
			wantOK:            true,
		},
		{
			name:              "preserves splitN behavior",
			id:                "domain-1/cert-1/extra",
			wantDomainID:      "domain-1",
			wantCertificateID: "cert-1/extra",
			wantOK:            true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotCertificateID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotCertificateID != tt.wantCertificateID {
				t.Fatalf("certificate ID = %q, want %q", gotCertificateID, tt.wantCertificateID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := CertificateModel{
		Name:          types.StringValue("JWT Signing Certificate"),
		Type:          types.StringValue("pkcs12-am-certificate"),
		Configuration: types.StringValue(`{"storepass":"changeit","keypass":"changeit","algorithm":"RS256"}`),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":          "JWT Signing Certificate",
		"type":          "pkcs12-am-certificate",
		"configuration": `{"storepass":"changeit","keypass":"changeit","algorithm":"RS256"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := CertificateModel{
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		Configuration: types.StringValue(`{"storepass":"real-secret","keypass":"real-secret"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new-name",
		"type":          "pkcs12-am-certificate",
		"configuration": `{"storepass":"********","keypass":"********"}`,
	})

	if got, want := model.Name.ValueString(), "new-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "pkcs12-am-certificate"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"storepass":"real-secret","keypass":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}
}

func TestCertificateCRUDPreservesSecretConfiguration(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})
	var bodies []map[string]interface{}
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		body := decodeCertificateBody(t, r)
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "cert-123", "name": body["name"], "type": body["type"]})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "cert-123",
				"name":          "read-certificate",
				"type":          "pkcs12-am-certificate",
				"configuration": `{"storepass":"********","keypass":"********"}`,
			})
		case http.MethodPut:
			body := decodeCertificateBody(t, r)
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "cert-123", "name": body["name"], "type": body["type"]})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &CertificateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := certificatePlan(t, schemaResp.Schema, CertificateModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("created-certificate"),
		Type:          types.StringValue("pkcs12-am-certificate"),
		Configuration: types.StringValue(`{"storepass":"real-secret","keypass":"real-secret"}`),
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
	var read CertificateModel
	if diags := readResp.State.Get(context.Background(), &read); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if read.Configuration.ValueString() != `{"storepass":"real-secret","keypass":"real-secret"}` {
		t.Fatalf("read configuration = %q", read.Configuration.ValueString())
	}

	updatePlan := certificatePlan(t, schemaResp.Schema, CertificateModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("updated-certificate"),
		Type:          types.StringValue("pkcs12-am-certificate"),
		Configuration: types.StringValue(`{"storepass":"updated-secret","keypass":"updated-secret"}`),
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
	wantBodies := []map[string]interface{}{
		{"name": "created-certificate", "type": "pkcs12-am-certificate", "configuration": `{"storepass":"real-secret","keypass":"real-secret"}`},
		{"name": "updated-certificate", "type": "pkcs12-am-certificate", "configuration": `{"storepass":"updated-secret","keypass":"updated-secret"}`},
	}
	if !reflect.DeepEqual(bodies, wantBodies) {
		t.Fatalf("bodies = %#v, want %#v", bodies, wantBodies)
	}
}

func TestCertificateReadRemovesMissingCertificateAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing certificate", statusCode: http.StatusNotFound, wantRemove: true},
		{name: "server error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &CertificateResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := certificateState(t, schemaResp.Schema, CertificateModel{
				ID:            types.StringValue("cert-123"),
				DomainID:      types.StringValue("domain-123"),
				Name:          types.StringValue("certificate"),
				Type:          types.StringValue("pkcs12-am-certificate"),
				Configuration: types.StringValue(`{"storepass":"real-secret"}`),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing certificate to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestCertificateReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			http.Error(w, "update failed", http.StatusInternalServerError)
		case http.MethodDelete:
			http.Error(w, "delete failed", http.StatusInternalServerError)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &CertificateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := certificatePlan(t, schemaResp.Schema, CertificateModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("certificate"),
		Type:          types.StringValue("pkcs12-am-certificate"),
		Configuration: types.StringValue(`{"storepass":"secret"}`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := certificateState(t, schemaResp.Schema, CertificateModel{
		ID:            types.StringValue("cert-123"),
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("certificate"),
		Type:          types.StringValue("pkcs12-am-certificate"),
		Configuration: types.StringValue(`{"storepass":"secret"}`),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestCertificateImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&CertificateResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestCertificateCreateReadUpdateAndDeleteReportInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &CertificateResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"domain_id":     tftypes.Number,
			"name":          tftypes.String,
			"type":          tftypes.String,
			"configuration": tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":            tftypes.NewValue(tftypes.String, "cert-123"),
			"domain_id":     tftypes.NewValue(tftypes.Number, 123),
			"name":          tftypes.NewValue(tftypes.String, "certificate"),
			"type":          tftypes.NewValue(tftypes.String, "pkcs12-am-certificate"),
			"configuration": tftypes.NewValue(tftypes.String, "{}"),
		},
	)

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
	}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestCertificateImportStateSetsDomainAndID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewCertificateResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: certificateState(t, schemaResp.Schema, CertificateModel{
		ID:            types.StringValue("placeholder"),
		DomainID:      types.StringValue("placeholder"),
		Name:          types.StringValue("certificate"),
		Type:          types.StringValue("pkcs12-am-certificate"),
		Configuration: types.StringValue(`{"storepass":"secret"}`),
	})}

	(&CertificateResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/cert-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported CertificateModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.ID.ValueString(), "cert-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func decodeCertificateBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func certificatePlan(t *testing.T, schema resourceschema.Schema, model CertificateModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func certificateState(t *testing.T, schema resourceschema.Schema, model CertificateModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
