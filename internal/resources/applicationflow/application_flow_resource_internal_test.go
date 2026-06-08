package applicationflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestApplicationFlowMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewApplicationFlowResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_application_flow" {
		t.Fatalf("type name = %q, want graviteeam_application_flow", resp.TypeName)
	}
}

func TestApplicationFlowSchemaDeclaresRequiredAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewApplicationFlowResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "application_id", "flows"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if !strings.Contains(resp.Schema.Description, "GET/PUT") {
		t.Fatalf("schema description = %q, want GET/PUT ownership hint", resp.Schema.Description)
	}
}

func TestApplicationFlowConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ApplicationFlowResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestApplicationFlowConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ApplicationFlowResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestApplicationFlowConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ApplicationFlowResource{}
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

func TestDecodeFlowsParsesCompleteApplicationFlowList(t *testing.T) {
	t.Parallel()

	diag := &fakeDiagnostics{}

	got, ok := decodeFlows(`[{"id":"login","name":"Login","enabled":true},{"id":"mfa","enabled":false}]`, diag)
	if !ok {
		t.Fatalf("expected decode success, got diagnostics %#v", diag.errors)
	}
	if len(got) != 2 {
		t.Fatalf("flows length = %d, want 2", len(got))
	}
	first, ok := got[0].(map[string]interface{})
	if !ok {
		t.Fatalf("first flow = %T, want map", got[0])
	}
	if first["id"] != "login" || first["name"] != "Login" || first["enabled"] != true {
		t.Fatalf("first flow = %#v", first)
	}
}

func TestDecodeFlowsRejectsMalformedOrNonListApplicationJSON(t *testing.T) {
	t.Parallel()

	for name, input := range map[string]string{
		"malformed": `{`,
		"object":    `{"id":"login"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			diag := &fakeDiagnostics{}
			got, ok := decodeFlows(input, diag)
			if ok {
				t.Fatalf("decode = %#v, want failure", got)
			}
			if len(diag.errors) != 1 {
				t.Fatalf("diagnostics = %#v, want one error", diag.errors)
			}
		})
	}
}

func TestDecodeFlowsAllowsExplicitEmptyApplicationList(t *testing.T) {
	t.Parallel()

	diag := &fakeDiagnostics{}

	got, ok := decodeFlows(`[]`, diag)
	if !ok {
		t.Fatalf("expected decode success, got diagnostics %#v", diag.errors)
	}
	if len(got) != 0 {
		t.Fatalf("flows = %#v, want empty", got)
	}
}

func TestEncodeFlowsProducesStableApplicationJSON(t *testing.T) {
	t.Parallel()

	got, err := encodeFlows([]interface{}{
		map[string]interface{}{"id": "login", "enabled": true},
	})
	if err != nil {
		t.Fatalf("encode flows: %v", err)
	}
	want := `[{"enabled":true,"id":"login"}]`
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestEncodeFlowsReturnsApplicationMarshalError(t *testing.T) {
	t.Parallel()

	_, err := encodeFlows([]interface{}{func() {}})
	if err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestFakeDiagnosticsRecordsApplicationSummaryAndDetail(t *testing.T) {
	t.Parallel()

	diag := &fakeDiagnostics{}
	diag.AddError("summary", "detail")

	want := []diagnosticError{{summary: "summary", detail: "detail"}}
	if !reflect.DeepEqual(diag.errors, want) {
		t.Fatalf("errors = %#v, want %#v", diag.errors, want)
	}
}

func TestApplicationFlowCRUDUsesCompleteFlowListPayloads(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})
	var putBodies [][]interface{}
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/flows", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "login", "enabled": true}})
		case http.MethodPut:
			var body []interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode application flow PUT body: %v", err)
			}
			putBodies = append(putBodies, body)
			_ = json.NewEncoder(w).Encode(body)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFlowResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := applicationFlowPlan(t, schemaResp.Schema, ApplicationFlowModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Flows:         types.StringValue(`[{"id":"login","enabled":true}]`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var created ApplicationFlowModel
	if diags := createResp.State.Get(context.Background(), &created); diags.HasError() {
		t.Fatalf("get created state: %#v", diags)
	}
	if created.Flows.ValueString() != `[{"enabled":true,"id":"login"}]` {
		t.Fatalf("created flows = %q", created.Flows.ValueString())
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}

	updatePlan := applicationFlowPlan(t, schemaResp.Schema, ApplicationFlowModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Flows:         types.StringValue(`[{"id":"mfa","enabled":false}]`),
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
	if len(putBodies[0]) != 1 || len(putBodies[1]) != 1 || len(putBodies[2]) != 0 {
		t.Fatalf("unexpected PUT bodies: %#v", putBodies)
	}
}

func TestApplicationFlowReadRemovesMissingApplication(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/flows", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFlowResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ApplicationFlowModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Flows:         types.StringValue(`[]`),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing application flows to remove state, got %#v", readResp.State.Raw)
	}
}

func TestApplicationFlowCRUDReportsRemoteErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/flows", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "remote error", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFlowResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := applicationFlowPlan(t, schemaResp.Schema, ApplicationFlowModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Flows:         types.StringValue(`[]`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: tfsdk.State(plan)}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: tfsdk.State(plan)}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: tfsdk.State(plan)}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestApplicationFlowDeleteIgnoresMissingApplication(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/applications/app-123/flows", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ApplicationFlowResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ApplicationFlowModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Flows:         types.StringValue(`[]`),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestApplicationFlowCreateAndUpdateRejectInvalidFlowsJSON(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ApplicationFlowResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := applicationFlowPlan(t, schemaResp.Schema, ApplicationFlowModel{
		DomainID:      types.StringValue("domain-123"),
		ApplicationID: types.StringValue("app-123"),
		Flows:         types.StringValue(`{"id":"login"}`),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: tfsdk.State(plan)}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}
}

func TestApplicationFlowCreateReadAndUpdateReportInvalidStateData(t *testing.T) {
	resourceUnderTest := &ApplicationFlowResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"domain_id":      tftypes.Number,
			"application_id": tftypes.String,
			"flows":          tftypes.String,
		}},
		map[string]tftypes.Value{
			"domain_id":      tftypes.NewValue(tftypes.Number, 123),
			"application_id": tftypes.NewValue(tftypes.String, "app-123"),
			"flows":          tftypes.NewValue(tftypes.String, `[]`),
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
}

func TestApplicationFlowImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&ApplicationFlowResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestApplicationFlowImportSetsDomainAndApplicationIDs(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewApplicationFlowResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &ApplicationFlowModel{}); diags.HasError() {
		t.Fatalf("set empty state: %#v", diags)
	}
	importResp := &resource.ImportStateResponse{State: state}

	NewApplicationFlowResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/app-123",
	}, importResp)

	if importResp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", importResp.Diagnostics)
	}
	var imported ApplicationFlowModel
	if diags := importResp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if imported.DomainID.ValueString() != "domain-123" || imported.ApplicationID.ValueString() != "app-123" {
		t.Fatalf("imported = %#v", imported)
	}
}

func applicationFlowPlan(t *testing.T, schema resourceschema.Schema, model ApplicationFlowModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

type fakeDiagnostics struct {
	errors []diagnosticError
}

type diagnosticError struct {
	summary string
	detail  string
}

func (d *fakeDiagnostics) AddError(summary string, detail string) {
	d.errors = append(d.errors, diagnosticError{summary: summary, detail: detail})
}
