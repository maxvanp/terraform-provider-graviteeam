package scope

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

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewScopeResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_scope"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewScopeResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "key", "name"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"description", "discovery", "expires_in", "icon_uri", "parameterized"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "discovery", "parameterized"} {
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
	(&ScopeResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ScopeResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ScopeResource{}
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

func TestBuildUpdateBodyClearsRemovedScopeFields(t *testing.T) {
	t.Parallel()

	plan := ScopeModel{
		Name:          types.StringValue("updated"),
		Discovery:     types.BoolValue(false),
		Parameterized: types.BoolValue(true),
	}
	state := ScopeModel{
		Name:          types.StringValue("updated"),
		Description:   types.StringValue("old description"),
		Discovery:     types.BoolValue(true),
		ExpiresIn:     types.Int64Value(3600),
		IconURI:       types.StringValue("https://example.test/icon.png"),
		Parameterized: types.BoolValue(false),
	}

	body := (&ScopeResource{}).buildUpdateBody(plan, state)

	if body["description"] != "" {
		t.Fatalf("description = %#v, want empty string", body["description"])
	}
	if body["expiresIn"] != 0 {
		t.Fatalf("expiresIn = %#v, want 0", body["expiresIn"])
	}
	if body["iconUri"] != nil {
		t.Fatalf("iconUri = %#v, want nil", body["iconUri"])
	}
}

func TestBuildUpdateBodySetsScopeOptionalFields(t *testing.T) {
	t.Parallel()

	plan := ScopeModel{
		Name:          types.StringValue("updated"),
		Description:   types.StringValue("new description"),
		Discovery:     types.BoolValue(true),
		ExpiresIn:     types.Int64Value(7200),
		IconURI:       types.StringValue("https://example.test/new-icon.png"),
		Parameterized: types.BoolValue(false),
	}

	body := (&ScopeResource{}).buildUpdateBody(plan, ScopeModel{})

	if body["description"] != "new description" {
		t.Fatalf("description = %#v", body["description"])
	}
	if body["expiresIn"] != int64(7200) {
		t.Fatalf("expiresIn = %#v", body["expiresIn"])
	}
	if body["iconUri"] != "https://example.test/new-icon.png" {
		t.Fatalf("iconUri = %#v", body["iconUri"])
	}
}

func TestReadIntoModelMapsScopeResponse(t *testing.T) {
	model := ScopeModel{}

	(&ScopeResource{}).readIntoModel(&model, map[string]interface{}{
		"id":            "scope-id",
		"key":           "scope-key",
		"name":          "scope-name",
		"description":   "description",
		"discovery":     false,
		"expiresIn":     float64(3600),
		"iconUri":       "https://example.test/icon.png",
		"parameterized": true,
	})

	if model.ID.ValueString() != "scope-id" {
		t.Fatalf("id = %q, want scope-id", model.ID.ValueString())
	}
	if model.Key.ValueString() != "scope-key" {
		t.Fatalf("key = %q, want scope-key", model.Key.ValueString())
	}
	if model.Name.ValueString() != "scope-name" {
		t.Fatalf("name = %q, want scope-name", model.Name.ValueString())
	}
	if model.Description.ValueString() != "description" {
		t.Fatalf("description = %q, want description", model.Description.ValueString())
	}
	if model.Discovery.ValueBool() {
		t.Fatalf("discovery = true, want false")
	}
	if model.ExpiresIn.ValueInt64() != 3600 {
		t.Fatalf("expiresIn = %d, want 3600", model.ExpiresIn.ValueInt64())
	}
	if model.IconURI.ValueString() != "https://example.test/icon.png" {
		t.Fatalf("iconUri = %q, want URL", model.IconURI.ValueString())
	}
	if !model.Parameterized.ValueBool() {
		t.Fatalf("parameterized = false, want true")
	}
}

func TestReadIntoModelHandlesEmptyAndIntegerOptionalScopeFields(t *testing.T) {
	t.Parallel()

	resource := &ScopeResource{}

	zeroModel := ScopeModel{
		ExpiresIn: types.Int64Null(),
	}
	resource.readIntoModel(&zeroModel, map[string]interface{}{
		"description": "",
		"expiresIn":   int64(0),
		"iconUri":     "",
	})
	if !zeroModel.Description.IsNull() || !zeroModel.ExpiresIn.IsNull() || !zeroModel.IconURI.IsNull() {
		t.Fatalf("zero model = %#v, want null optional fields", zeroModel)
	}

	integerModel := ScopeModel{}
	resource.readIntoModel(&integerModel, map[string]interface{}{
		"expiresIn": int64(7200),
	})
	if integerModel.ExpiresIn.ValueInt64() != 7200 {
		t.Fatalf("expiresIn = %#v, want 7200", integerModel.ExpiresIn)
	}

	missingModel := ScopeModel{
		ExpiresIn: types.Int64Value(3600),
	}
	resource.readIntoModel(&missingModel, map[string]interface{}{})
	if !missingModel.ExpiresIn.IsNull() {
		t.Fatalf("missing expiresIn = %#v, want null", missingModel.ExpiresIn)
	}
}

func TestScopeCRUDClearsRemovedOptionalFields(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/scopes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            "scope-123",
			"key":           body["key"],
			"name":          body["name"],
			"description":   body["description"],
			"discovery":     body["discovery"],
			"expiresIn":     body["expiresIn"],
			"iconUri":       body["iconUri"],
			"parameterized": body["parameterized"],
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/scopes/scope-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "scope-123",
				"key":           "claim_scope",
				"name":          "Claim scope",
				"description":   "created",
				"discovery":     true,
				"expiresIn":     float64(3600),
				"iconUri":       "https://example.test/icon.png",
				"parameterized": false,
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "scope-123",
				"key":           "claim_scope",
				"name":          body["name"],
				"description":   body["description"],
				"discovery":     body["discovery"],
				"expiresIn":     body["expiresIn"],
				"iconUri":       body["iconUri"],
				"parameterized": body["parameterized"],
			})
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ScopeResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := scopePlan(t, schemaResp.Schema, ScopeModel{
		DomainID:      types.StringValue("domain-123"),
		Key:           types.StringValue("claim_scope"),
		Name:          types.StringValue("Claim scope"),
		Description:   types.StringValue("created"),
		Discovery:     types.BoolValue(true),
		ExpiresIn:     types.Int64Value(3600),
		IconURI:       types.StringValue("https://example.test/icon.png"),
		Parameterized: types.BoolValue(false),
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

	updatePlan := scopePlan(t, schemaResp.Schema, ScopeModel{
		DomainID:      types.StringValue("domain-123"),
		Key:           types.StringValue("claim_scope"),
		Name:          types.StringValue("Claim scope updated"),
		Description:   types.StringNull(),
		Discovery:     types.BoolValue(false),
		ExpiresIn:     types.Int64Null(),
		IconURI:       types.StringNull(),
		Parameterized: types.BoolValue(true),
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

	if !reflect.DeepEqual(methods, []string{"create", "read", "update", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if got := bodies[0]["expiresIn"]; got != float64(3600) {
		t.Fatalf("create expiresIn = %#v, want 3600", got)
	}
	if got := bodies[1]["description"]; got != "" {
		t.Fatalf("update description = %#v, want clear string", got)
	}
	if got := bodies[1]["expiresIn"]; got != float64(0) {
		t.Fatalf("update expiresIn = %#v, want clear zero", got)
	}
	if got := bodies[1]["iconUri"]; got != nil {
		t.Fatalf("update iconUri = %#v, want nil", got)
	}
}

func TestScopeReadRemovesMissingScopeAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing scope", statusCode: http.StatusNotFound, wantRemove: true},
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/scopes/scope-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &ScopeResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := scopeState(t, schemaResp.Schema, ScopeModel{
				ID:            types.StringValue("scope-123"),
				DomainID:      types.StringValue("domain-123"),
				Key:           types.StringValue("claim_scope"),
				Name:          types.StringValue("Claim scope"),
				Discovery:     types.BoolValue(true),
				Parameterized: types.BoolValue(false),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing scope to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestScopeReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/scopes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/scopes/scope-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &ScopeResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := scopePlan(t, schemaResp.Schema, ScopeModel{
		DomainID:      types.StringValue("domain-123"),
		Key:           types.StringValue("claim_scope"),
		Name:          types.StringValue("Claim scope"),
		Discovery:     types.BoolValue(true),
		Parameterized: types.BoolValue(false),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := scopeState(t, schemaResp.Schema, ScopeModel{
		ID:            types.StringValue("scope-123"),
		DomainID:      types.StringValue("domain-123"),
		Key:           types.StringValue("claim_scope"),
		Name:          types.StringValue("Claim scope"),
		Discovery:     types.BoolValue(true),
		Parameterized: types.BoolValue(false),
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

func TestScopeDeleteReportsInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ScopeResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: invalidScopeRaw()},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestScopeCreateAndReadReportInvalidRequestData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ScopeResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: invalidScopeRaw()},
	}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected invalid create plan diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: invalidScopeRaw()},
	}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected invalid read state diagnostics")
	}
}

func TestScopeUpdateReportsInvalidPlanAndStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ScopeResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	valid := ScopeModel{
		ID:            types.StringValue("scope-123"),
		DomainID:      types.StringValue("domain-123"),
		Key:           types.StringValue("claim_scope"),
		Name:          types.StringValue("Claim scope"),
		Discovery:     types.BoolValue(true),
		Parameterized: types.BoolValue(false),
	}

	invalidPlanResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: invalidScopeRaw()},
		State: scopeState(t, schemaResp.Schema, valid),
	}, invalidPlanResp)
	if !invalidPlanResp.Diagnostics.HasError() {
		t.Fatal("expected invalid plan diagnostics")
	}

	invalidStateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  scopePlan(t, schemaResp.Schema, valid),
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: invalidScopeRaw()},
	}, invalidStateResp)
	if !invalidStateResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestScopeDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/scopes/scope-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ScopeResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := scopeState(t, schemaResp.Schema, ScopeModel{
		ID:            types.StringValue("scope-123"),
		DomainID:      types.StringValue("domain-123"),
		Key:           types.StringValue("claim_scope"),
		Name:          types.StringValue("Claim scope"),
		Discovery:     types.BoolValue(true),
		Parameterized: types.BoolValue(false),
	})

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestScopeImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&ScopeResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestScopeImportStateSetsAttributes(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&ScopeResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: scopeState(t, schemaResp.Schema, ScopeModel{
		ID:            types.StringValue("old-scope"),
		DomainID:      types.StringValue("old-domain"),
		Key:           types.StringValue("claim_scope"),
		Name:          types.StringValue("Claim scope"),
		Discovery:     types.BoolValue(true),
		Parameterized: types.BoolValue(false),
	})}

	(&ScopeResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/scope-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state ScopeModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	if got, want := state.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain_id = %q, want %q", got, want)
	}
	if got, want := state.ID.ValueString(), "scope-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func scopePlan(t *testing.T, schema resourceschema.Schema, model ScopeModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func scopeState(t *testing.T, schema resourceschema.Schema, model ScopeModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func invalidScopeRaw() tftypes.Value {
	return tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"domain_id":     tftypes.Number,
			"key":           tftypes.String,
			"name":          tftypes.String,
			"description":   tftypes.String,
			"discovery":     tftypes.Bool,
			"expires_in":    tftypes.Number,
			"icon_uri":      tftypes.String,
			"parameterized": tftypes.Bool,
		}},
		map[string]tftypes.Value{
			"id":            tftypes.NewValue(tftypes.String, "scope-123"),
			"domain_id":     tftypes.NewValue(tftypes.Number, 123),
			"key":           tftypes.NewValue(tftypes.String, "claim_scope"),
			"name":          tftypes.NewValue(tftypes.String, "Claim scope"),
			"description":   tftypes.NewValue(tftypes.String, nil),
			"discovery":     tftypes.NewValue(tftypes.Bool, true),
			"expires_in":    tftypes.NewValue(tftypes.Number, nil),
			"icon_uri":      tftypes.NewValue(tftypes.String, nil),
			"parameterized": tftypes.NewValue(tftypes.Bool, false),
		},
	)
}
