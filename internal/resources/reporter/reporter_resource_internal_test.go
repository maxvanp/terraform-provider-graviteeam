package reporter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewReporterResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_reporter"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewReporterResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "configuration", true, false, false)
	assertBoolAttribute(t, resp.Schema.Attributes, "enabled", false, true, true)
	assertBoolAttribute(t, resp.Schema.Attributes, "inherited", false, true, true)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ReporterResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestBuildBodyUsesPlannedReporterFields(t *testing.T) {
	t.Parallel()

	plan := ReporterModel{
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{"directory":"/tmp"}`),
		Enabled:       types.BoolValue(true),
		Inherited:     types.BoolValue(false),
	}

	got := buildBody(plan, map[string]interface{}{"inherited": true})
	want := map[string]interface{}{
		"name":          "file reporter",
		"type":          "reporter-am-file",
		"configuration": `{"directory":"/tmp"}`,
		"enabled":       true,
		"inherited":     false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyPreservesCurrentInheritedWhenPlanUnknown(t *testing.T) {
	t.Parallel()

	plan := ReporterModel{
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{}`),
		Enabled:       types.BoolValue(false),
		Inherited:     types.BoolUnknown(),
	}

	got := buildBody(plan, map[string]interface{}{"inherited": true})

	if got["inherited"] != true {
		t.Fatalf("inherited = %#v, want preserved true", got["inherited"])
	}
}

func TestReadIntoModelMapsReporterFieldsAndPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := ReporterModel{
		Configuration: types.StringValue(`{"secret":"planned"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "updated",
		"type":          "reporter-am-file",
		"configuration": `{"secret":"***"}`,
		"enabled":       false,
		"inherited":     true,
	})

	if model.Name.ValueString() != "updated" ||
		model.Type.ValueString() != "reporter-am-file" ||
		model.Enabled.ValueBool() ||
		!model.Inherited.ValueBool() {
		t.Fatalf("model fields not mapped: %#v", model)
	}
	if model.Configuration.ValueString() != `{"secret":"planned"}` {
		t.Fatalf("configuration = %q, want preserved", model.Configuration.ValueString())
	}
}

func TestReporterCRUDUsesCurrentStateForUpdateBody(t *testing.T) {
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
	var methods []string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/reporters", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected reporter collection method %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		methods = append(methods, "create")
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "reporter-123"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/reporters/reporter-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "reporter-123",
				"name":      "file reporter",
				"type":      "reporter-am-file",
				"enabled":   true,
				"inherited": true,
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "reporter-123"})
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected reporter item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ReporterResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := reporterPlan(t, schemaResp.Schema, ReporterModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{"directory":"/tmp"}`),
		Enabled:       types.BoolValue(true),
		Inherited:     types.BoolValue(false),
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

	updatePlan := reporterPlan(t, schemaResp.Schema, ReporterModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("file reporter updated"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{"directory":"/var/log"}`),
		Enabled:       types.BoolValue(false),
		Inherited:     types.BoolUnknown(),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: readResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if want := []string{"create", "read", "read", "update", "delete"}; !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
	if got, want := bodies[1]["inherited"], true; got != want {
		t.Fatalf("update inherited = %#v, want preserved %#v", got, want)
	}
	if got, want := bodies[1]["configuration"], `{"directory":"/var/log"}`; got != want {
		t.Fatalf("update configuration = %#v, want %#v", got, want)
	}
}

func TestReporterReadRemovesMissingReporterAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{
			name:       "missing reporter",
			statusCode: http.StatusNotFound,
			wantRemove: true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantRemove: false,
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/reporters/reporter-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &ReporterResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := reporterState(t, schemaResp.Schema, ReporterModel{
				ID:            types.StringValue("reporter-123"),
				DomainID:      types.StringValue("domain-123"),
				Name:          types.StringValue("file reporter"),
				Type:          types.StringValue("reporter-am-file"),
				Configuration: types.StringValue(`{"directory":"/tmp"}`),
				Enabled:       types.BoolValue(true),
				Inherited:     types.BoolValue(false),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing reporter to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestReporterReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/reporters", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/reporters/reporter-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "reporter-123",
				"name":      "file reporter",
				"type":      "reporter-am-file",
				"enabled":   true,
				"inherited": true,
			})
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

	resourceUnderTest := &ReporterResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := reporterPlan(t, schemaResp.Schema, ReporterModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{"directory":"/tmp"}`),
		Enabled:       types.BoolValue(true),
		Inherited:     types.BoolValue(false),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := reporterState(t, schemaResp.Schema, ReporterModel{
		ID:            types.StringValue("reporter-123"),
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{"directory":"/tmp"}`),
		Enabled:       types.BoolValue(true),
		Inherited:     types.BoolUnknown(),
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

func TestReporterUpdateReportsPreReadError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/reporters/reporter-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "read before update failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ReporterResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := reporterPlan(t, schemaResp.Schema, ReporterModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{"directory":"/tmp"}`),
		Enabled:       types.BoolValue(true),
		Inherited:     types.BoolValue(false),
	})
	state := reporterState(t, schemaResp.Schema, ReporterModel{
		ID:            types.StringValue("reporter-123"),
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{"directory":"/tmp"}`),
		Enabled:       types.BoolValue(true),
		Inherited:     types.BoolValue(false),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected update pre-read diagnostics")
	}
}

func TestReporterImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&ReporterResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestReporterConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ReporterResource{}
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

func TestReporterImportStateSetsDomainAndID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewReporterResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: reporterState(t, schemaResp.Schema, ReporterModel{
		ID:            types.StringValue("placeholder"),
		DomainID:      types.StringValue("placeholder"),
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{"directory":"/tmp"}`),
		Enabled:       types.BoolValue(true),
		Inherited:     types.BoolValue(false),
	})}

	(&ReporterResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/reporter-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported ReporterModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.ID.ValueString(), "reporter-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func reporterPlan(t *testing.T, schema resourceschema.Schema, model ReporterModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func reporterState(t *testing.T, schema resourceschema.Schema, model ReporterModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
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

func assertBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.BoolAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
