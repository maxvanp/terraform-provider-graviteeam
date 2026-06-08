package generatedcertificate

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

func TestGeneratedCertificateMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewGeneratedCertificateResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_generated_certificate" {
		t.Fatalf("type name = %q, want graviteeam_generated_certificate", resp.TypeName)
	}
}

func TestGeneratedCertificateSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewGeneratedCertificateResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["domain_id"]; !attr.IsRequired() {
		t.Fatal("domain_id should be required")
	}
	if attr := resp.Schema.Attributes["rotation_trigger"]; !attr.IsOptional() {
		t.Fatal("rotation_trigger should be optional")
	}
	for _, name := range []string{"id", "name", "type"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
}

func TestGeneratedCertificateConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GeneratedCertificateResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestGeneratedCertificateConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GeneratedCertificateResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestGeneratedCertificateConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &GeneratedCertificateResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if resourceUnderTest.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestGeneratedCertificateUpdateIsNoOp(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewGeneratedCertificateResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &GeneratedCertificateModel{
		ID:              types.StringValue("cert-123"),
		DomainID:        types.StringValue("domain-123"),
		RotationTrigger: types.StringValue("initial"),
		Name:            types.StringValue("generated-cert"),
		Type:            types.StringValue("pem"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	resp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	NewGeneratedCertificateResource().Update(context.Background(), resource.UpdateRequest{State: state}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected update diagnostics: %#v", resp.Diagnostics)
	}
	var updated GeneratedCertificateModel
	if diags := resp.State.Get(context.Background(), &updated); diags.HasError() {
		t.Fatalf("get update state: %#v", diags)
	}
	if got, want := updated.ID.ValueString(), "cert-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func TestGeneratedCertificateCreateAndUpdateReportInvalidStateData(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewGeneratedCertificateResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":               tftypes.String,
			"domain_id":        tftypes.Number,
			"rotation_trigger": tftypes.String,
			"name":             tftypes.String,
			"type":             tftypes.String,
			"metadata":         tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":               tftypes.NewValue(tftypes.String, nil),
			"domain_id":        tftypes.NewValue(tftypes.Number, 123),
			"rotation_trigger": tftypes.NewValue(tftypes.String, "rotate"),
			"name":             tftypes.NewValue(tftypes.String, nil),
			"type":             tftypes.NewValue(tftypes.String, nil),
			"metadata":         tftypes.NewValue(tftypes.String, nil),
		},
	)

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	NewGeneratedCertificateResource().Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
	}, &createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	NewGeneratedCertificateResource().Update(context.Background(), resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, &updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := resource.DeleteResponse{}
	NewGeneratedCertificateResource().Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, &deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestReadGeneratedCertificateMapsReturnedFields(t *testing.T) {
	t.Parallel()

	model := GeneratedCertificateModel{}

	readGeneratedCertificate(&model, map[string]interface{}{
		"name": "system-generated-cert",
		"type": "pem",
	})

	if got, want := model.Name.ValueString(), "system-generated-cert"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "pem"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
}

func TestReadGeneratedCertificatePreservesMissingFields(t *testing.T) {
	t.Parallel()

	model := GeneratedCertificateModel{
		Name: types.StringValue("existing-name"),
		Type: types.StringValue("existing-type"),
	}

	readGeneratedCertificate(&model, map[string]interface{}{})

	if got, want := model.Name.ValueString(), "existing-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "existing-type"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
}

func TestCertificateIDReturnsID(t *testing.T) {
	t.Parallel()

	got, ok := certificateID(map[string]interface{}{"id": "cert-id"})
	if !ok {
		t.Fatalf("expected id")
	}
	if got != "cert-id" {
		t.Fatalf("id = %q, want cert-id", got)
	}
}

func TestCertificateIDRejectsMissingEmptyOrMalformedID(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]interface{}{
		"missing":   {},
		"empty":     {"id": ""},
		"malformed": {"id": 42},
	}

	for name, result := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if id, ok := certificateID(result); ok {
				t.Fatalf("id = %q, want not ok", id)
			}
		})
	}
}

func TestGeneratedCertificateCRUDUsesRotateReadDelete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	var methods []string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/rotate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected rotate method %s", r.Method)
		}
		methods = append(methods, "rotate")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "cert-123",
			"name": "generated-cert",
			"type": "pem",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "cert-123",
				"name": "generated-cert-read",
				"type": "pem",
			})
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected certificate item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GeneratedCertificateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := generatedCertificatePlan(t, schemaResp.Schema, GeneratedCertificateModel{
		DomainID:        types.StringValue("domain-123"),
		RotationTrigger: types.StringValue("initial"),
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
	var readState GeneratedCertificateModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := readState.Name.ValueString(), "generated-cert-read"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: readResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if want := []string{"rotate", "read", "delete"}; !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
}

func TestGeneratedCertificateReadRemovesMissingCertificateAndDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GeneratedCertificateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &GeneratedCertificateModel{
		ID:              types.StringValue("cert-123"),
		DomainID:        types.StringValue("domain-123"),
		Name:            types.StringValue("generated-cert"),
		Type:            types.StringValue("pem"),
		RotationTrigger: types.StringValue("initial"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing generated certificate to remove state, got %#v", readResp.State.Raw)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestGeneratedCertificateCreateReportsMissingRotatedCertificateID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/rotate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		_, _ = w.Write([]byte(`{"name":"generated-cert","type":"pem"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GeneratedCertificateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := generatedCertificatePlan(t, schemaResp.Schema, GeneratedCertificateModel{
		DomainID:        types.StringValue("domain-123"),
		RotationTrigger: types.StringValue("initial"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected missing certificate id diagnostics")
	}
}

func TestGeneratedCertificateReadAndDeleteReportNonNotFoundErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			http.Error(w, "read failed", http.StatusInternalServerError)
		case http.MethodDelete:
			http.Error(w, "delete failed", http.StatusInternalServerError)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &GeneratedCertificateResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &GeneratedCertificateModel{
		ID:              types.StringValue("cert-123"),
		DomainID:        types.StringValue("domain-123"),
		Name:            types.StringValue("generated-cert"),
		Type:            types.StringValue("pem"),
		RotationTrigger: types.StringValue("initial"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read error diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete error diagnostics")
	}
}

func TestGeneratedCertificateImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&GeneratedCertificateResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestGeneratedCertificateImportStateSetsAttributes(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&GeneratedCertificateResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	if diags := resp.State.Set(context.Background(), &GeneratedCertificateModel{
		ID:              types.StringValue("old-cert"),
		DomainID:        types.StringValue("old-domain"),
		RotationTrigger: types.StringNull(),
		Name:            types.StringValue("generated-cert"),
		Type:            types.StringValue("pem"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	(&GeneratedCertificateResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/cert-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state GeneratedCertificateModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	if got, want := state.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain_id = %q, want %q", got, want)
	}
	if got, want := state.ID.ValueString(), "cert-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func generatedCertificatePlan(t *testing.T, schema resourceschema.Schema, model GeneratedCertificateModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
