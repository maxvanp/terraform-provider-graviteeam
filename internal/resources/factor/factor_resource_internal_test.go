package factor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestFactorMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewFactorResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_factor" {
		t.Fatalf("type name = %q, want graviteeam_factor", resp.TypeName)
	}
}

func TestFactorSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewFactorResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "factor_type"} {
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

func TestFactorConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &FactorResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := FactorModel{
		Name:       types.StringValue("Login TOTP"),
		FactorType: types.StringValue("TOTP"),
	}

	got := buildCreateBody(plan, "otp-am-factor")
	want := map[string]interface{}{
		"name":          "Login TOTP",
		"type":          "otp-am-factor",
		"factorType":    "TOTP",
		"configuration": "{}",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyOmitsImmutableFactorType(t *testing.T) {
	t.Parallel()

	plan := FactorModel{
		Name:       types.StringValue("Updated TOTP"),
		FactorType: types.StringValue("TOTP"),
	}

	got := buildUpdateBody(plan, "otp-am-factor")
	want := map[string]interface{}{
		"name":          "Updated TOTP",
		"type":          "otp-am-factor",
		"configuration": "{}",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
	if _, ok := got["factorType"]; ok {
		t.Fatalf("factorType should be omitted from update body")
	}
}

func TestReadIntoModelMapsLowercaseAPIFactorType(t *testing.T) {
	t.Parallel()

	model := FactorModel{}

	readIntoModel(&model, map[string]interface{}{
		"name":       "Login Email",
		"factorType": "email",
	})

	if got, want := model.Name.ValueString(), "Login Email"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.FactorType.ValueString(), "EMAIL"; got != want {
		t.Fatalf("factorType = %q, want %q", got, want)
	}
}

func TestReadIntoModelFallsBackToPluginType(t *testing.T) {
	t.Parallel()

	model := FactorModel{}

	readIntoModel(&model, map[string]interface{}{
		"type": "sms-am-factor",
	})

	if got, want := model.FactorType.ValueString(), "SMS"; got != want {
		t.Fatalf("factorType = %q, want %q", got, want)
	}
}

func TestReadIntoModelPreservesUnknownAPIFactorType(t *testing.T) {
	t.Parallel()

	model := FactorModel{}

	readIntoModel(&model, map[string]interface{}{
		"factorType": "custom",
	})

	if got, want := model.FactorType.ValueString(), "custom"; got != want {
		t.Fatalf("factorType = %q, want %q", got, want)
	}
}

func TestFactorTypeValidator(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"TOTP", "EMAIL", "SMS"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			resp := validator.StringResponse{}
			factorTypeValidator{}.ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(value),
			}, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("expected %s to be accepted, got %v", value, resp.Diagnostics)
			}
		})
	}
}

func TestFactorTypeValidatorRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	resp := validator.StringResponse{}
	factorTypeValidator{}.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringValue("PUSH"),
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected unknown factor type to be rejected")
	}
}

func TestFactorCRUDUsesPluginTypeAndOmitsFactorTypeOnUpdate(t *testing.T) {
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
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/factors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected factor collection method %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		methods = append(methods, "create")
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "factor-123"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/factors/factor-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         "factor-123",
				"name":       "Login TOTP",
				"factorType": "otp",
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "factor-123"})
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected factor item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FactorResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := factorPlan(t, schemaResp.Schema, FactorModel{
		DomainID:   types.StringValue("domain-123"),
		Name:       types.StringValue("Login TOTP"),
		FactorType: types.StringValue("TOTP"),
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

	updatePlan := factorPlan(t, schemaResp.Schema, FactorModel{
		DomainID:   types.StringValue("domain-123"),
		Name:       types.StringValue("Login TOTP Updated"),
		FactorType: types.StringValue("TOTP"),
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

	wantCreate := map[string]interface{}{
		"name":          "Login TOTP",
		"type":          "otp-am-factor",
		"factorType":    "TOTP",
		"configuration": "{}",
	}
	if !reflect.DeepEqual(bodies[0], wantCreate) {
		t.Fatalf("create body = %#v, want %#v", bodies[0], wantCreate)
	}
	if _, ok := bodies[1]["factorType"]; ok {
		t.Fatalf("update body should omit factorType: %#v", bodies[1])
	}
	if want := []string{"create", "read", "update", "delete"}; !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
}

func TestFactorReadRemovesMissingFactorAndDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/factors/factor-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &FactorResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &FactorModel{
		ID:         types.StringValue("factor-123"),
		DomainID:   types.StringValue("domain-123"),
		Name:       types.StringValue("Login TOTP"),
		FactorType: types.StringValue("TOTP"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing factor to remove state, got %#v", readResp.State.Raw)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestFactorReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/factors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/factors/factor-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			http.Error(w, "read failed", http.StatusInternalServerError)
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

	resourceUnderTest := &FactorResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := factorPlan(t, schemaResp.Schema, FactorModel{
		DomainID:   types.StringValue("domain-123"),
		Name:       types.StringValue("Login TOTP"),
		FactorType: types.StringValue("TOTP"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := factorState(t, schemaResp.Schema, FactorModel{
		ID:         types.StringValue("factor-123"),
		DomainID:   types.StringValue("domain-123"),
		Name:       types.StringValue("Login TOTP"),
		FactorType: types.StringValue("TOTP"),
	})
	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

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

func TestFactorImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&FactorResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func factorPlan(t *testing.T, schema resourceschema.Schema, model FactorModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func factorState(t *testing.T, schema resourceschema.Schema, model FactorModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
