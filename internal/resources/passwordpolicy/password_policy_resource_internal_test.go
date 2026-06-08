package passwordpolicy

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
	NewPasswordPolicyResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_password_policy"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewPasswordPolicyResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{
		"min_length",
		"max_length",
		"max_consecutive_letters",
		"expiry_duration",
		"old_passwords",
		"include_numbers",
		"include_special_characters",
		"letters_in_mixed_case",
		"exclude_passwords_in_dictionary",
		"exclude_user_profile_info_in_password",
		"password_history_enabled",
		"default_policy",
	} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Fatalf("attribute %q should be optional+computed", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; attr == nil || !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&PasswordPolicyResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		id                   string
		wantDomainID         string
		wantPasswordPolicyID string
		wantOK               bool
	}{
		{
			name:                 "valid",
			id:                   "domain-1/policy-1",
			wantDomainID:         "domain-1",
			wantPasswordPolicyID: "policy-1",
			wantOK:               true,
		},
		{
			name:                 "preserves existing splitN behavior",
			id:                   "domain-1/policy-1/extra",
			wantDomainID:         "domain-1",
			wantPasswordPolicyID: "policy-1/extra",
			wantOK:               true,
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

			gotDomainID, gotPasswordPolicyID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotPasswordPolicyID != tt.wantPasswordPolicyID {
				t.Fatalf("password policy ID = %q, want %q", gotPasswordPolicyID, tt.wantPasswordPolicyID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	resource := &PasswordPolicyResource{}
	plan := PasswordPolicyModel{
		Name:                             types.StringValue("Strict Policy"),
		MinLength:                        types.Int64Value(12),
		MaxLength:                        types.Int64Value(64),
		MaxConsecutiveLetters:            types.Int64Value(3),
		ExpiryDuration:                   types.Int64Value(3600),
		OldPasswords:                     types.Int64Value(5),
		IncludeNumbers:                   types.BoolValue(true),
		IncludeSpecialCharacters:         types.BoolValue(false),
		LettersInMixedCase:               types.BoolValue(true),
		ExcludePasswordsInDictionary:     types.BoolValue(false),
		ExcludeUserProfileInfoInPassword: types.BoolValue(true),
		PasswordHistoryEnabled:           types.BoolValue(true),
		DefaultPolicy:                    types.BoolValue(false),
	}

	got := resource.buildBody(plan, true)
	want := map[string]interface{}{
		"name":                             "Strict Policy",
		"minLength":                        int64(12),
		"maxLength":                        int64(64),
		"maxConsecutiveLetters":            int64(3),
		"expiryDuration":                   int64(3600),
		"oldPasswords":                     int64(5),
		"includeNumbers":                   true,
		"includeSpecialCharacters":         false,
		"lettersInMixedCase":               true,
		"excludePasswordsInDictionary":     false,
		"excludeUserProfileInfoInPassword": true,
		"passwordHistoryEnabled":           true,
		"defaultPolicy":                    false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyOmitsDefaultPolicyWhenCreateUsesDedicatedEndpoint(t *testing.T) {
	t.Parallel()

	resource := &PasswordPolicyResource{}
	plan := PasswordPolicyModel{
		Name:          types.StringValue("Default Policy"),
		DefaultPolicy: types.BoolValue(true),
		MinLength:     types.Int64Null(),
	}

	got := resource.buildBody(plan, false)
	if _, ok := got["defaultPolicy"]; ok {
		t.Fatalf("defaultPolicy should be omitted from create body: %#v", got)
	}
	if _, ok := got["minLength"]; ok {
		t.Fatalf("null minLength should be omitted from body: %#v", got)
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	resource := &PasswordPolicyResource{}
	model := PasswordPolicyModel{
		DomainID: types.StringValue("domain-1"),
		Name:     types.StringValue("old-name"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":                               "policy-1",
		"name":                             "Strict Policy",
		"minLength":                        float64(12),
		"maxLength":                        int64(64),
		"maxConsecutiveLetters":            float64(3),
		"expiryDuration":                   int64(3600),
		"oldPasswords":                     float64(5),
		"includeNumbers":                   true,
		"includeSpecialCharacters":         false,
		"lettersInMixedCase":               true,
		"excludePasswordsInDictionary":     false,
		"excludeUserProfileInfoInPassword": true,
		"passwordHistoryEnabled":           true,
		"defaultPolicy":                    false,
		"ignoredUnsupportedNumericRepresentation": int(2),
	})

	assertString(t, model.ID, "id", "policy-1")
	assertString(t, model.Name, "name", "Strict Policy")
	assertInt64(t, model.MinLength, "minLength", 12)
	assertInt64(t, model.MaxLength, "maxLength", 64)
	assertInt64(t, model.MaxConsecutiveLetters, "maxConsecutiveLetters", 3)
	assertInt64(t, model.ExpiryDuration, "expiryDuration", 3600)
	assertInt64(t, model.OldPasswords, "oldPasswords", 5)
	assertBool(t, model.IncludeNumbers, "includeNumbers", true)
	assertBool(t, model.IncludeSpecialCharacters, "includeSpecialCharacters", false)
	assertBool(t, model.LettersInMixedCase, "lettersInMixedCase", true)
	assertBool(t, model.ExcludePasswordsInDictionary, "excludePasswordsInDictionary", false)
	assertBool(t, model.ExcludeUserProfileInfoInPassword, "excludeUserProfileInfoInPassword", true)
	assertBool(t, model.PasswordHistoryEnabled, "passwordHistoryEnabled", true)
	assertBool(t, model.DefaultPolicy, "defaultPolicy", false)
}

func TestReadMissingValuesAsNull(t *testing.T) {
	t.Parallel()

	resource := &PasswordPolicyResource{}
	model := PasswordPolicyModel{}

	resource.readIntoModel(&model, map[string]interface{}{"name": "Sparse Policy"})

	if !model.MinLength.IsNull() {
		t.Fatalf("missing minLength should be null")
	}
	if !model.IncludeNumbers.IsNull() {
		t.Fatalf("missing includeNumbers should be null")
	}
}

func TestPasswordPolicyCRUDUsesDedicatedDefaultEndpoint(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(passwordPolicyResponse("policy-123", body, false))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                       "policy-123",
				"name":                     "Strict Policy",
				"minLength":                float64(12),
				"includeNumbers":           true,
				"passwordHistoryEnabled":   true,
				"defaultPolicy":            true,
				"includeSpecialCharacters": true,
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			defaultPolicy, _ := body["defaultPolicy"].(bool)
			_ = json.NewEncoder(w).Encode(passwordPolicyResponse("policy-123", body, defaultPolicy))
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123/default", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("default method = %s", r.Method)
		}
		methods = append(methods, "default")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                       "policy-123",
			"name":                     "Strict Policy",
			"minLength":                float64(12),
			"includeNumbers":           true,
			"passwordHistoryEnabled":   true,
			"defaultPolicy":            true,
			"includeSpecialCharacters": true,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &PasswordPolicyResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := passwordPolicyPlan(t, schemaResp.Schema, PasswordPolicyModel{
		DomainID:               types.StringValue("domain-123"),
		Name:                   types.StringValue("Strict Policy"),
		MinLength:              types.Int64Value(12),
		IncludeNumbers:         types.BoolValue(true),
		PasswordHistoryEnabled: types.BoolValue(true),
		DefaultPolicy:          types.BoolValue(true),
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

	updatePlan := passwordPolicyPlan(t, schemaResp.Schema, PasswordPolicyModel{
		DomainID:               types.StringValue("domain-123"),
		Name:                   types.StringValue("Strict Policy updated"),
		MinLength:              types.Int64Value(10),
		IncludeNumbers:         types.BoolValue(false),
		PasswordHistoryEnabled: types.BoolValue(false),
		DefaultPolicy:          types.BoolValue(false),
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

	if !reflect.DeepEqual(methods, []string{"create", "default", "read", "update", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if _, ok := bodies[0]["defaultPolicy"]; ok {
		t.Fatalf("create body should not include defaultPolicy: %#v", bodies[0])
	}
	if got := bodies[1]["defaultPolicy"]; got != false {
		t.Fatalf("update defaultPolicy = %#v, want false", got)
	}
}

func TestPasswordPolicyReadRemovesMissingPolicyAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing policy", statusCode: http.StatusNotFound, wantRemove: true},
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &PasswordPolicyResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := passwordPolicyState(t, schemaResp.Schema, PasswordPolicyModel{
				ID:       types.StringValue("policy-123"),
				DomainID: types.StringValue("domain-123"),
				Name:     types.StringValue("Strict Policy"),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing password policy to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestPasswordPolicyReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			http.Error(w, "update failed", http.StatusInternalServerError)
		case http.MethodDelete:
			http.Error(w, "delete failed", http.StatusInternalServerError)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &PasswordPolicyResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := passwordPolicyPlan(t, schemaResp.Schema, PasswordPolicyModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("Strict Policy"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := passwordPolicyState(t, schemaResp.Schema, PasswordPolicyModel{
		ID:       types.StringValue("policy-123"),
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("Strict Policy"),
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

func TestPasswordPolicyCreateReportsDefaultEndpointError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "policy-123",
			"name": "Strict Policy",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123/default", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("default method = %s, want POST", r.Method)
		}
		http.Error(w, "set default failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &PasswordPolicyResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := passwordPolicyPlan(t, schemaResp.Schema, PasswordPolicyModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("Strict Policy"),
		DefaultPolicy: types.BoolValue(true),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected default endpoint diagnostics")
	}
}

func TestPasswordPolicyUpdateReportsDefaultEndpointError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("item method = %s, want PUT", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "policy-123",
			"name": "Strict Policy",
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/policy-123/default", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("default method = %s, want POST", r.Method)
		}
		http.Error(w, "set default failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &PasswordPolicyResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := passwordPolicyPlan(t, schemaResp.Schema, PasswordPolicyModel{
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("Strict Policy"),
		DefaultPolicy: types.BoolValue(true),
	})
	state := passwordPolicyState(t, schemaResp.Schema, PasswordPolicyModel{
		ID:       types.StringValue("policy-123"),
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("Strict Policy"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected default endpoint diagnostics")
	}
}

func TestPasswordPolicyImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&PasswordPolicyResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func passwordPolicyResponse(id string, body map[string]interface{}, defaultPolicy bool) map[string]interface{} {
	result := map[string]interface{}{
		"id":            id,
		"name":          body["name"],
		"defaultPolicy": defaultPolicy,
	}
	for _, key := range []string{
		"minLength",
		"maxLength",
		"maxConsecutiveLetters",
		"expiryDuration",
		"oldPasswords",
		"includeNumbers",
		"includeSpecialCharacters",
		"lettersInMixedCase",
		"excludePasswordsInDictionary",
		"excludeUserProfileInfoInPassword",
		"passwordHistoryEnabled",
	} {
		if value, ok := body[key]; ok {
			result[key] = value
		}
	}
	return result
}

func passwordPolicyPlan(t *testing.T, schema resourceschema.Schema, model PasswordPolicyModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func passwordPolicyState(t *testing.T, schema resourceschema.Schema, model PasswordPolicyModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func assertString(t *testing.T, value types.String, name string, want string) {
	t.Helper()
	if got := value.ValueString(); got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}

func assertInt64(t *testing.T, value types.Int64, name string, want int64) {
	t.Helper()
	if got := value.ValueInt64(); got != want {
		t.Fatalf("%s = %d, want %d", name, got, want)
	}
}

func assertBool(t *testing.T, value types.Bool, name string, want bool) {
	t.Helper()
	if got := value.ValueBool(); got != want {
		t.Fatalf("%s = %t, want %t", name, got, want)
	}
}
