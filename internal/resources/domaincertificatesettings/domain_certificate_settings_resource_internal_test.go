package domaincertificatesettings

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

func TestDomainCertificateSettingsMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewDomainCertificateSettingsResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_domain_certificate_settings" {
		t.Fatalf("type name = %q, want graviteeam_domain_certificate_settings", resp.TypeName)
	}
}

func TestDomainCertificateSettingsSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewDomainCertificateSettingsResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "fallback_certificate_id"} {
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

func TestDomainCertificateSettingsConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &DomainCertificateSettingsResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildBodySetsFallbackCertificate(t *testing.T) {
	t.Parallel()

	got := buildBody("cert-1")
	want := map[string]interface{}{"fallbackCertificate": "cert-1"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildDeleteBodyClearsFallbackCertificate(t *testing.T) {
	t.Parallel()

	got := buildDeleteBody()

	if value, ok := got["fallbackCertificate"]; !ok || value != nil {
		t.Fatalf("fallbackCertificate = %#v, want nil", got)
	}
}

func TestReadFallbackCertificateID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		domain map[string]interface{}
		want   string
	}{
		"present": {
			domain: map[string]interface{}{
				"certificateSettings": map[string]interface{}{
					"fallbackCertificate": "cert-1",
				},
			},
			want: "cert-1",
		},
		"missing settings": {
			domain: map[string]interface{}{},
			want:   "",
		},
		"wrong settings type": {
			domain: map[string]interface{}{"certificateSettings": "invalid"},
			want:   "",
		},
		"missing fallback": {
			domain: map[string]interface{}{"certificateSettings": map[string]interface{}{}},
			want:   "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := readFallbackCertificateID(test.domain); got != test.want {
				t.Fatalf("fallback = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDomainCertificateSettingsCRUDPreservesSingleFieldOwnership(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificate-settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT certificate settings, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode certificate settings body: %v", err)
		}
		putBodies = append(putBodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"updated": true})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET domain, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "domain-123",
			"certificateSettings": map[string]interface{}{
				"fallbackCertificate": "cert-read",
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &DomainCertificateSettingsResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := domainCertificateSettingsPlan(t, schemaResp.Schema, DomainCertificateSettingsModel{
		DomainID:              types.StringValue("domain-123"),
		FallbackCertificateID: types.StringValue("cert-create"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var created DomainCertificateSettingsModel
	if diags := createResp.State.Get(context.Background(), &created); diags.HasError() {
		t.Fatalf("get created state: %#v", diags)
	}
	if created.ID.ValueString() != "domain-123" {
		t.Fatalf("created id = %q", created.ID.ValueString())
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var read DomainCertificateSettingsModel
	if diags := readResp.State.Get(context.Background(), &read); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if read.FallbackCertificateID.ValueString() != "cert-read" {
		t.Fatalf("read fallback = %q", read.FallbackCertificateID.ValueString())
	}

	updatePlan := domainCertificateSettingsPlan(t, schemaResp.Schema, DomainCertificateSettingsModel{
		DomainID:              types.StringValue("domain-123"),
		FallbackCertificateID: types.StringValue("cert-update"),
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
	if !reflect.DeepEqual(putBodies[0], map[string]interface{}{"fallbackCertificate": "cert-create"}) {
		t.Fatalf("create body = %#v", putBodies[0])
	}
	if !reflect.DeepEqual(putBodies[1], map[string]interface{}{"fallbackCertificate": "cert-update"}) {
		t.Fatalf("update body = %#v", putBodies[1])
	}
	if value, ok := putBodies[2]["fallbackCertificate"]; !ok || value != nil {
		t.Fatalf("delete body = %#v", putBodies[2])
	}
}

func TestDomainCertificateSettingsReadRemovesMissingDomainOrFallbackAndDeleteIgnores404(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       map[string]interface{}
	}{
		{
			name:       "missing domain",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "missing fallback",
			statusCode: http.StatusOK,
			body:       map[string]interface{}{"id": "domain-123", "certificateSettings": map[string]interface{}{}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("domain method = %s, want GET", r.Method)
				}
				if tt.statusCode != http.StatusOK {
					http.Error(w, "not found", tt.statusCode)
					return
				}
				_ = json.NewEncoder(w).Encode(tt.body)
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificate-settings", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Fatalf("certificate settings method = %s, want PUT", r.Method)
				}
				http.Error(w, "not found", http.StatusNotFound)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &DomainCertificateSettingsResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := tfsdk.State{Schema: schemaResp.Schema}
			if diags := state.Set(context.Background(), &DomainCertificateSettingsModel{
				ID:                    types.StringValue("domain-123"),
				DomainID:              types.StringValue("domain-123"),
				FallbackCertificateID: types.StringValue("cert-123"),
			}); diags.HasError() {
				t.Fatalf("set state: %#v", diags)
			}

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
			}
			if !readResp.State.Raw.IsNull() {
				t.Fatalf("expected missing certificate settings to remove state, got %#v", readResp.State.Raw)
			}

			deleteResp := &resource.DeleteResponse{}
			resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
			if deleteResp.Diagnostics.HasError() {
				t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
			}
		})
	}
}

func TestDomainCertificateSettingsReadReportsServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("domain method = %s, want GET", r.Method)
		}
		http.Error(w, "read failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &DomainCertificateSettingsResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := domainCertificateSettingsState(t, schemaResp.Schema, DomainCertificateSettingsModel{
		ID:                    types.StringValue("domain-123"),
		DomainID:              types.StringValue("domain-123"),
		FallbackCertificateID: types.StringValue("cert-123"),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}
}

func TestDomainCertificateSettingsReportsWriteErrors(t *testing.T) {
	tests := []struct {
		name      string
		operation string
	}{
		{name: "create", operation: "create"},
		{name: "update", operation: "update"},
		{name: "delete", operation: "delete"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificate-settings", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Fatalf("certificate settings method = %s, want PUT", r.Method)
				}
				http.Error(w, tt.operation+" failed", http.StatusInternalServerError)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &DomainCertificateSettingsResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			plan := domainCertificateSettingsPlan(t, schemaResp.Schema, DomainCertificateSettingsModel{
				DomainID:              types.StringValue("domain-123"),
				FallbackCertificateID: types.StringValue("cert-123"),
			})
			state := domainCertificateSettingsState(t, schemaResp.Schema, DomainCertificateSettingsModel{
				ID:                    types.StringValue("domain-123"),
				DomainID:              types.StringValue("domain-123"),
				FallbackCertificateID: types.StringValue("cert-123"),
			})

			switch tt.operation {
			case "create":
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
				resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected create diagnostics")
				}
			case "update":
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
				resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected update diagnostics")
				}
			case "delete":
				resp := &resource.DeleteResponse{}
				resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected delete diagnostics")
				}
			}
		})
	}
}

func domainCertificateSettingsPlan(t *testing.T, schema resourceschema.Schema, model DomainCertificateSettingsModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func domainCertificateSettingsState(t *testing.T, schema resourceschema.Schema, model DomainCertificateSettingsModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
