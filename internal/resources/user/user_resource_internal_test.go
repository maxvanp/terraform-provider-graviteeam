package user

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
	NewUserResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_user"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewUserResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "username"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{
		"email",
		"first_name",
		"last_name",
		"display_name",
		"force_reset_password",
		"enabled",
		"locked",
		"pre_registration",
		"reset_password",
		"reset_password_trigger",
		"registration_confirmation_trigger",
	} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "force_reset_password", "enabled", "locked", "pre_registration"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
	if attr := resp.Schema.Attributes["reset_password"]; attr == nil || !attr.IsSensitive() {
		t.Fatalf("reset_password should be sensitive")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&UserResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&UserResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &UserResource{}
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

func TestBuildMergedUpdateBodyPreservesUnmanagedCurrentFields(t *testing.T) {
	t.Parallel()

	current := map[string]interface{}{
		"id":                    "ignored",
		"accountNonExpired":     true,
		"accountNonLocked":      false,
		"additionalInformation": map[string]interface{}{"team": "platform"},
		"client":                "app",
		"createdAt":             "2026-01-01T00:00:00Z",
		"credentialsNonExpired": true,
		"displayName":           "Old Display",
		"email":                 "old@example.com",
		"enabled":               true,
		"externalId":            "external-1",
		"firstName":             "Old",
		"forceResetPassword":    false,
		"lastName":              "Name",
		"loggedAt":              "2026-01-02T00:00:00Z",
		"loginsCount":           float64(4),
		"preRegistration":       true,
		"preferredLanguage":     "fr",
		"registrationCompleted": false,
		"source":                "gravitee",
		"updatedAt":             "2026-01-03T00:00:00Z",
		"unexpected":            "must-not-leak",
	}
	plan := UserModel{
		Email:              types.StringValue("new@example.com"),
		FirstName:          types.StringValue("New"),
		LastName:           types.StringValue("Person"),
		DisplayName:        types.StringValue("New Display"),
		ForceResetPassword: types.BoolValue(true),
		PreRegistration:    types.BoolValue(false),
	}

	got := buildMergedUpdateBody(current, plan)

	for _, field := range []string{
		"accountNonExpired",
		"accountNonLocked",
		"additionalInformation",
		"client",
		"createdAt",
		"credentialsNonExpired",
		"enabled",
		"externalId",
		"loggedAt",
		"loginsCount",
		"preferredLanguage",
		"registrationCompleted",
		"source",
		"updatedAt",
	} {
		if !reflect.DeepEqual(got[field], current[field]) {
			t.Fatalf("%s = %#v, want preserved %#v", field, got[field], current[field])
		}
	}
	if got["email"] != "new@example.com" ||
		got["firstName"] != "New" ||
		got["lastName"] != "Person" ||
		got["displayName"] != "New Display" {
		t.Fatalf("planned profile fields not applied: %#v", got)
	}
	if got["forceResetPassword"] != true || got["preRegistration"] != false {
		t.Fatalf("planned booleans not applied: %#v", got)
	}
	if _, ok := got["unexpected"]; ok {
		t.Fatalf("unexpected field leaked into update body: %#v", got)
	}
}

func TestBuildMergedUpdateBodyKeepsCurrentOptionalFieldsWhenPlanNull(t *testing.T) {
	t.Parallel()

	current := map[string]interface{}{
		"email":       "old@example.com",
		"firstName":   "Old",
		"lastName":    "Name",
		"displayName": "Old Display",
	}
	plan := UserModel{
		Email:              types.StringNull(),
		FirstName:          types.StringUnknown(),
		LastName:           types.StringNull(),
		DisplayName:        types.StringUnknown(),
		ForceResetPassword: types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
	}

	got := buildMergedUpdateBody(current, plan)

	for _, field := range []string{"email", "firstName", "lastName", "displayName"} {
		if got[field] != current[field] {
			t.Fatalf("%s = %#v, want preserved %#v", field, got[field], current[field])
		}
	}
}

func TestCopyUserUpdateFieldsOnlyCopiesAllowedFields(t *testing.T) {
	t.Parallel()

	current := map[string]interface{}{
		"email":      "alice@example.com",
		"enabled":    true,
		"unexpected": "ignored",
	}

	got := copyUserUpdateFields(current)

	if !reflect.DeepEqual(got, map[string]interface{}{"email": "alice@example.com", "enabled": true}) {
		t.Fatalf("copied fields = %#v", got)
	}
}

func TestShouldTrigger(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		plan types.String
		prev types.String
		want bool
	}{
		"null plan does not trigger":         {types.StringNull(), types.StringValue("old"), false},
		"unknown plan does not trigger":      {types.StringUnknown(), types.StringValue("old"), false},
		"new trigger runs":                   {types.StringValue("new"), types.StringNull(), true},
		"changed trigger runs":               {types.StringValue("new"), types.StringValue("old"), true},
		"unchanged trigger does not run":     {types.StringValue("same"), types.StringValue("same"), false},
		"reset wrapper delegates to trigger": {types.StringValue("new"), types.StringValue("old"), true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := shouldTrigger(test.plan, test.prev)
			if name == "reset wrapper delegates to trigger" {
				got = shouldResetPassword(test.plan, test.prev)
			}
			if got != test.want {
				t.Fatalf("trigger decision = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReadIntoModelMapsAPIFieldsAndLockState(t *testing.T) {
	t.Parallel()

	resource := &UserResource{}
	model := UserModel{
		Email:       types.StringValue("previous@example.com"),
		FirstName:   types.StringValue("Previous"),
		LastName:    types.StringValue("Name"),
		DisplayName: types.StringValue("Previous Name"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":                 "user-1",
		"username":           "alice",
		"email":              "alice@example.com",
		"firstName":          "Alice",
		"lastName":           "Liddell",
		"displayName":        "Alice Liddell",
		"forceResetPassword": true,
		"enabled":            true,
		"accountNonLocked":   false,
		"preRegistration":    false,
	})

	if model.ID.ValueString() != "user-1" ||
		model.Username.ValueString() != "alice" ||
		model.Email.ValueString() != "alice@example.com" ||
		model.FirstName.ValueString() != "Alice" ||
		model.LastName.ValueString() != "Liddell" ||
		model.DisplayName.ValueString() != "Alice Liddell" {
		t.Fatalf("string fields not mapped: %#v", model)
	}
	if !model.ForceResetPassword.ValueBool() || !model.Enabled.ValueBool() || !model.Locked.ValueBool() || model.PreRegistration.ValueBool() {
		t.Fatalf("boolean fields not mapped: %#v", model)
	}
}

func TestReadIntoModelClearsMissingOptionalStrings(t *testing.T) {
	t.Parallel()

	resource := &UserResource{}
	model := UserModel{
		Email:       types.StringValue("previous@example.com"),
		FirstName:   types.StringValue("Previous"),
		LastName:    types.StringValue("Name"),
		DisplayName: types.StringValue("Previous Name"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":       "user-1",
		"username": "alice",
	})

	if !model.Email.IsNull() || !model.FirstName.IsNull() || !model.LastName.IsNull() || !model.DisplayName.IsNull() {
		t.Fatalf("optional strings were not cleared: %#v", model)
	}
}

func TestUserCRUDUsesMergedProfileUpdateAndSeparateActions(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string
	readCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
			"username":           body["username"],
			"email":              body["email"],
			"firstName":          body["firstName"],
			"lastName":           body["lastName"],
			"forceResetPassword": body["forceResetPassword"],
			"preRegistration":    body["preRegistration"],
			"enabled":            body["enabled"],
			"accountNonLocked":   true,
		}))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			readCount++
			response := map[string]interface{}{
				"username":              "alice",
				"email":                 "alice@example.com",
				"firstName":             "Alice",
				"lastName":              "Liddell",
				"displayName":           "Alice Liddell",
				"forceResetPassword":    false,
				"preRegistration":       true,
				"enabled":               true,
				"accountNonExpired":     true,
				"accountNonLocked":      true,
				"additionalInformation": map[string]interface{}{"team": "platform"},
				"credentialsNonExpired": true,
				"externalId":            "external-1",
				"preferredLanguage":     "fr",
				"registrationCompleted": false,
				"source":                "gravitee",
			}
			if readCount >= 3 {
				response["username"] = "alice-updated"
				response["email"] = "alice-updated@example.com"
				response["lastName"] = "Updated"
				response["displayName"] = "Alice Updated"
				response["forceResetPassword"] = true
				response["preRegistration"] = false
				response["enabled"] = false
				response["accountNonLocked"] = true
			} else if readCount == 2 {
				response["accountNonLocked"] = false
			}
			_ = json.NewEncoder(w).Encode(userResponse("user-123", response))
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(userResponse("user-123", body))
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("status method = %s, want PUT", r.Method)
		}
		methods = append(methods, "status")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode status body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
			"username":           "alice",
			"enabled":            body["enabled"],
			"accountNonLocked":   true,
			"forceResetPassword": false,
			"preRegistration":    true,
		}))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/username", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("username method = %s, want PATCH", r.Method)
		}
		methods = append(methods, "username")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode username body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
			"username": body["username"],
		}))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/lock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("lock method = %s, want POST", r.Method)
		}
		methods = append(methods, "lock")
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/unlock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unlock method = %s, want POST", r.Method)
		}
		methods = append(methods, "unlock")
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/resetPassword", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("reset method = %s, want POST", r.Method)
		}
		methods = append(methods, "reset")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode reset body: %v", err)
		}
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/sendRegistrationConfirmation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("registration method = %s, want POST", r.Method)
		}
		methods = append(methods, "registration")
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := userPlan(t, schemaResp.Schema, UserModel{
		DomainID:           types.StringValue("domain-123"),
		Username:           types.StringValue("alice"),
		Email:              types.StringValue("alice@example.com"),
		FirstName:          types.StringValue("Alice"),
		LastName:           types.StringValue("Liddell"),
		DisplayName:        types.StringValue("Alice Liddell"),
		ForceResetPassword: types.BoolValue(false),
		Enabled:            types.BoolValue(true),
		Locked:             types.BoolValue(true),
		PreRegistration:    types.BoolValue(true),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}

	updatePlan := userPlan(t, schemaResp.Schema, UserModel{
		DomainID:            types.StringValue("domain-123"),
		Username:            types.StringValue("alice-updated"),
		Email:               types.StringValue("alice-updated@example.com"),
		FirstName:           types.StringValue("Alice"),
		LastName:            types.StringValue("Updated"),
		DisplayName:         types.StringValue("Alice Updated"),
		ForceResetPassword:  types.BoolValue(true),
		Enabled:             types.BoolValue(false),
		Locked:              types.BoolValue(false),
		PreRegistration:     types.BoolValue(false),
		ResetPassword:       types.StringValue("rotated-secret"),
		ResetTrigger:        types.StringValue("rotation-1"),
		RegistrationTrigger: types.StringValue("send-1"),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: createResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	var updateState UserModel
	if diags := updateResp.State.Get(context.Background(), &updateState); diags.HasError() {
		t.Fatalf("get update state: %#v", diags)
	}
	if updateState.ResetPassword.ValueString() != "rotated-secret" {
		t.Fatalf("reset_password after update = %q", updateState.ResetPassword.ValueString())
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if !reflect.DeepEqual(methods, []string{"create", "read", "update", "lock", "read", "username", "read", "update", "status", "unlock", "reset", "registration", "read", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if got := bodies[0]["enabled"]; got != true {
		t.Fatalf("create enabled = %#v", got)
	}
	createUpdateBody := bodies[1]
	if createUpdateBody["displayName"] != "Alice Liddell" {
		t.Fatalf("post-create update body = %#v", createUpdateBody)
	}
	for _, field := range []string{"accountNonExpired", "accountNonLocked", "additionalInformation", "credentialsNonExpired", "externalId", "preferredLanguage", "registrationCompleted", "source"} {
		if _, ok := createUpdateBody[field]; !ok {
			t.Fatalf("post-create merged body missing preserved field %q: %#v", field, createUpdateBody)
		}
	}
	if got := bodies[2]["username"]; got != "alice-updated" {
		t.Fatalf("username body = %#v", bodies[2])
	}
	updateBody := bodies[3]
	for _, field := range []string{"accountNonExpired", "accountNonLocked", "additionalInformation", "credentialsNonExpired", "externalId", "preferredLanguage", "registrationCompleted", "source"} {
		if _, ok := updateBody[field]; !ok {
			t.Fatalf("update merged body missing preserved field %q: %#v", field, updateBody)
		}
	}
	if updateBody["email"] != "alice-updated@example.com" || updateBody["displayName"] != "Alice Updated" || updateBody["lastName"] != "Updated" {
		t.Fatalf("update body did not apply plan fields: %#v", updateBody)
	}
	if updateBody["forceResetPassword"] != true || updateBody["preRegistration"] != false {
		t.Fatalf("update body did not apply plan booleans: %#v", updateBody)
	}
	if got := bodies[4]["enabled"]; got != false {
		t.Fatalf("status body = %#v", bodies[4])
	}
	if got := bodies[5]["password"]; got != "rotated-secret" {
		t.Fatalf("reset password body = %#v", bodies[5])
	}
}

func TestUserUpdateNoChangesOnlyRefreshesState(t *testing.T) {
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected user method %s", r.Method)
		}
		methods = append(methods, "read")
		_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
			"username":           "alice",
			"email":              "alice@example.com",
			"firstName":          "Alice",
			"lastName":           "Liddell",
			"displayName":        "Alice Liddell",
			"enabled":            true,
			"accountNonLocked":   true,
			"forceResetPassword": false,
			"preRegistration":    true,
		}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	model := UserModel{
		ID:                 types.StringValue("user-123"),
		DomainID:           types.StringValue("domain-123"),
		Username:           types.StringValue("alice"),
		Email:              types.StringValue("alice@example.com"),
		FirstName:          types.StringValue("Alice"),
		LastName:           types.StringValue("Liddell"),
		DisplayName:        types.StringValue("Alice Liddell"),
		ForceResetPassword: types.BoolValue(false),
		Enabled:            types.BoolValue(true),
		Locked:             types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  userPlan(t, schemaResp.Schema, model),
		State: userState(t, schemaResp.Schema, model),
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	if !reflect.DeepEqual(methods, []string{"read"}) {
		t.Fatalf("methods = %#v, want final read only", methods)
	}
}

func TestUserUpdateReportsProfileUpdateErrorAfterSuccessfulRead(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
				"username":           "alice",
				"email":              "alice@example.com",
				"enabled":            true,
				"accountNonLocked":   true,
				"forceResetPassword": false,
				"preRegistration":    true,
			}))
		case http.MethodPut:
			http.Error(w, "profile update failed", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected user method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := baseUserModel()
	plan.Email = types.StringValue("updated@example.com")
	state := baseUserModel()
	state.Email = types.StringValue("alice@example.com")

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  userPlan(t, schemaResp.Schema, plan),
		State: userState(t, schemaResp.Schema, state),
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected profile update diagnostics")
	}
}

func TestUserUpdateLocksUserAndRefreshesState(t *testing.T) {
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/lock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("lock method = %s, want POST", r.Method)
		}
		methods = append(methods, "lock")
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected user method %s", r.Method)
		}
		methods = append(methods, "read")
		_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
			"username":           "alice",
			"enabled":            true,
			"accountNonLocked":   false,
			"forceResetPassword": false,
			"preRegistration":    true,
		}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := baseUserModel()
	plan.Locked = types.BoolValue(true)
	state := baseUserModel()

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  userPlan(t, schemaResp.Schema, plan),
		State: userState(t, schemaResp.Schema, state),
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	if !reflect.DeepEqual(methods, []string{"lock", "read"}) {
		t.Fatalf("methods = %#v, want lock then read", methods)
	}
	var updated UserModel
	if diags := updateResp.State.Get(context.Background(), &updated); diags.HasError() {
		t.Fatalf("get update state: %#v", diags)
	}
	if !updated.Locked.ValueBool() {
		t.Fatalf("locked = %#v, want true", updated.Locked)
	}
}

func TestUserReadRemovesMissingUserAndDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &UserModel{
		ID:                  types.StringValue("user-123"),
		DomainID:            types.StringValue("domain-123"),
		Username:            types.StringValue("alice"),
		Email:               types.StringValue("alice@example.com"),
		FirstName:           types.StringValue("Alice"),
		LastName:            types.StringValue("Liddell"),
		DisplayName:         types.StringValue("Alice Liddell"),
		ForceResetPassword:  types.BoolValue(false),
		Enabled:             types.BoolValue(true),
		Locked:              types.BoolValue(false),
		PreRegistration:     types.BoolValue(true),
		ResetPassword:       types.StringValue("secret"),
		ResetTrigger:        types.StringValue("rotation-1"),
		RegistrationTrigger: types.StringValue("registration-1"),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing user to remove state, got %#v", readResp.State.Raw)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestUserReadMapsRemoteUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                 "user-123",
			"username":           "alice",
			"email":              "alice@example.com",
			"firstName":          "Alice",
			"lastName":           "Liddell",
			"displayName":        "Alice Liddell",
			"forceResetPassword": true,
			"enabled":            true,
			"accountNonLocked":   false,
			"preRegistration":    false,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &UserResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := userState(t, schemaResp.Schema, UserModel{
		ID:                  types.StringValue("user-123"),
		DomainID:            types.StringValue("domain-123"),
		Username:            types.StringValue("old"),
		ResetPassword:       types.StringValue("secret"),
		ResetTrigger:        types.StringValue("rotation-1"),
		RegistrationTrigger: types.StringValue("registration-1"),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState UserModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if readState.Username.ValueString() != "alice" || readState.DisplayName.ValueString() != "Alice Liddell" {
		t.Fatalf("read state = %#v", readState)
	}
	if !readState.Locked.ValueBool() || !readState.ForceResetPassword.ValueBool() || readState.PreRegistration.ValueBool() {
		t.Fatalf("boolean fields = %#v", readState)
	}
	if readState.ResetPassword.ValueString() != "secret" ||
		readState.ResetTrigger.ValueString() != "rotation-1" ||
		readState.RegistrationTrigger.ValueString() != "registration-1" {
		t.Fatalf("action-only fields not preserved: %#v", readState)
	}
}

func TestUserCRUDReportsRemoteErrors(t *testing.T) {
	tests := map[string]struct {
		createStatus int
		itemStatus   int
		statusStatus int
		lockStatus   int
		action       func(context.Context, *UserResource, tfsdk.Plan, tfsdk.State, resourceschema.Schema) bool
	}{
		"create": {
			createStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *UserResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"post_create_read": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *UserResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"create_lock": {
			lockStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *UserResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"read": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *UserResource, _ tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
				r.Read(ctx, resource.ReadRequest{State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"update_username": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *UserResource, plan tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"update_status": {
			statusStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *UserResource, plan tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"delete": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *UserResource, _ tfsdk.Plan, state tfsdk.State, _ resourceschema.Schema) bool {
				resp := &resource.DeleteResponse{}
				r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("collection method = %s, want POST", r.Method)
				}
				if tc.createStatus != 0 {
					http.Error(w, "remote error", tc.createStatus)
					return
				}
				_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
					"username":           "alice",
					"enabled":            true,
					"accountNonLocked":   true,
					"forceResetPassword": false,
					"preRegistration":    true,
				}))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123", func(w http.ResponseWriter, r *http.Request) {
				if tc.itemStatus != 0 {
					http.Error(w, "remote error", tc.itemStatus)
					return
				}
				switch r.Method {
				case http.MethodGet, http.MethodPut:
					_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
						"username":           "alice",
						"enabled":            true,
						"accountNonLocked":   true,
						"forceResetPassword": false,
						"preRegistration":    true,
					}))
				case http.MethodDelete:
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Fatalf("item method = %s", r.Method)
				}
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/username", func(w http.ResponseWriter, _ *http.Request) {
				if tc.itemStatus != 0 {
					http.Error(w, "remote error", tc.itemStatus)
					return
				}
				_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{"username": "alice-updated"}))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/status", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Fatalf("status method = %s, want PUT", r.Method)
				}
				if tc.statusStatus != 0 {
					http.Error(w, "remote error", tc.statusStatus)
					return
				}
				_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
					"username":           "alice",
					"enabled":            false,
					"accountNonLocked":   true,
					"forceResetPassword": false,
					"preRegistration":    true,
				}))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/lock", func(w http.ResponseWriter, _ *http.Request) {
				if tc.lockStatus != 0 {
					http.Error(w, "remote error", tc.lockStatus)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &UserResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			planModel := UserModel{
				ID:                 types.StringValue("user-123"),
				DomainID:           types.StringValue("domain-123"),
				Username:           types.StringValue("alice-updated"),
				DisplayName:        types.StringValue("Alice Updated"),
				ForceResetPassword: types.BoolValue(false),
				Enabled:            types.BoolValue(false),
				Locked:             types.BoolValue(true),
				PreRegistration:    types.BoolValue(true),
			}
			stateModel := planModel
			stateModel.Username = types.StringValue("alice")
			stateModel.DisplayName = types.StringNull()
			stateModel.Enabled = types.BoolValue(true)
			stateModel.Locked = types.BoolValue(false)
			plan := userPlan(t, schemaResp.Schema, planModel)
			state := userState(t, schemaResp.Schema, stateModel)

			if !tc.action(context.Background(), resourceUnderTest, plan, state, schemaResp.Schema) {
				t.Fatal("expected diagnostics")
			}
		})
	}
}

func TestUserUpdateRequiresResetPasswordWhenTriggerChanges(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &UserResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := userPlan(t, schemaResp.Schema, UserModel{
		ID:                 types.StringValue("user-123"),
		DomainID:           types.StringValue("domain-123"),
		Username:           types.StringValue("alice"),
		ForceResetPassword: types.BoolValue(false),
		Enabled:            types.BoolValue(true),
		Locked:             types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
		ResetPassword:      types.StringNull(),
		ResetTrigger:       types.StringValue("rotation-2"),
	})
	state := userState(t, schemaResp.Schema, UserModel{
		ID:                 types.StringValue("user-123"),
		DomainID:           types.StringValue("domain-123"),
		Username:           types.StringValue("alice"),
		ForceResetPassword: types.BoolValue(false),
		Enabled:            types.BoolValue(true),
		Locked:             types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
		ResetTrigger:       types.StringValue("rotation-1"),
	})

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected missing reset password diagnostics")
	}
}

func TestUserUpdateReportsActionErrors(t *testing.T) {
	tests := map[string]struct {
		unlockStatus       int
		resetStatus        int
		registrationStatus int
		finalReadStatus    int
		plan               func() UserModel
		state              func() UserModel
	}{
		"unlock": {
			unlockStatus: http.StatusInternalServerError,
			plan: func() UserModel {
				model := baseUserModel()
				model.Locked = types.BoolValue(false)
				return model
			},
			state: func() UserModel {
				model := baseUserModel()
				model.Locked = types.BoolValue(true)
				return model
			},
		},
		"reset_password": {
			resetStatus: http.StatusInternalServerError,
			plan: func() UserModel {
				model := baseUserModel()
				model.ResetPassword = types.StringValue("rotated-secret")
				model.ResetTrigger = types.StringValue("rotation-2")
				return model
			},
			state: func() UserModel {
				model := baseUserModel()
				model.ResetTrigger = types.StringValue("rotation-1")
				return model
			},
		},
		"registration_confirmation": {
			registrationStatus: http.StatusInternalServerError,
			plan: func() UserModel {
				model := baseUserModel()
				model.RegistrationTrigger = types.StringValue("send-2")
				return model
			},
			state: func() UserModel {
				model := baseUserModel()
				model.RegistrationTrigger = types.StringValue("send-1")
				return model
			},
		},
		"final_read": {
			finalReadStatus: http.StatusInternalServerError,
			plan: func() UserModel {
				model := baseUserModel()
				model.ResetPassword = types.StringValue("rotated-secret")
				model.ResetTrigger = types.StringValue("rotation-2")
				return model
			},
			state: func() UserModel {
				model := baseUserModel()
				model.ResetTrigger = types.StringValue("rotation-1")
				return model
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("item method = %s, want GET", r.Method)
				}
				if tc.finalReadStatus != 0 {
					http.Error(w, "remote error", tc.finalReadStatus)
					return
				}
				_ = json.NewEncoder(w).Encode(userResponse("user-123", map[string]interface{}{
					"username":           "alice",
					"enabled":            true,
					"accountNonLocked":   true,
					"forceResetPassword": false,
					"preRegistration":    true,
				}))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/unlock", func(w http.ResponseWriter, _ *http.Request) {
				if tc.unlockStatus != 0 {
					http.Error(w, "remote error", tc.unlockStatus)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/resetPassword", func(w http.ResponseWriter, _ *http.Request) {
				if tc.resetStatus != 0 {
					http.Error(w, "remote error", tc.resetStatus)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/sendRegistrationConfirmation", func(w http.ResponseWriter, _ *http.Request) {
				if tc.registrationStatus != 0 {
					http.Error(w, "remote error", tc.registrationStatus)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &UserResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

			updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
				Plan:  userPlan(t, schemaResp.Schema, tc.plan()),
				State: userState(t, schemaResp.Schema, tc.state()),
			}, updateResp)
			if !updateResp.Diagnostics.HasError() {
				t.Fatal("expected update diagnostics")
			}
		})
	}
}

func TestUserCreateReadUpdateAndDeleteReportInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &UserResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := userInvalidRaw()

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
		State: userState(t, schemaResp.Schema, baseUserModel()),
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected invalid update plan diagnostics")
	}

	updateResp = &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  userPlan(t, schemaResp.Schema, baseUserModel()),
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected invalid update state diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestUserImportStateRejectsInvalidID(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewUserResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	importResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	NewUserResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "user-only",
	}, importResp)

	if !importResp.Diagnostics.HasError() {
		t.Fatal("expected invalid import diagnostics")
	}
}

func TestUserImportStateSetsAttributes(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewUserResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	importResp := &resource.ImportStateResponse{State: userState(t, schemaResp.Schema, baseUserModel())}

	NewUserResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/user-123",
	}, importResp)

	if importResp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", importResp.Diagnostics)
	}
	var state UserModel
	if diags := importResp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get import state: %#v", diags)
	}
	if got, want := state.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain_id = %q, want %q", got, want)
	}
	if got, want := state.ID.ValueString(), "user-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func baseUserModel() UserModel {
	return UserModel{
		ID:                  types.StringValue("user-123"),
		DomainID:            types.StringValue("domain-123"),
		Username:            types.StringValue("alice"),
		Email:               types.StringNull(),
		FirstName:           types.StringNull(),
		LastName:            types.StringNull(),
		DisplayName:         types.StringNull(),
		ForceResetPassword:  types.BoolValue(false),
		Enabled:             types.BoolValue(true),
		Locked:              types.BoolValue(false),
		PreRegistration:     types.BoolValue(true),
		ResetPassword:       types.StringNull(),
		ResetTrigger:        types.StringNull(),
		RegistrationTrigger: types.StringNull(),
	}
}

func userResponse(id string, body map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"id": id,
	}
	for key, value := range body {
		result[key] = value
	}
	return result
}

func userState(t *testing.T, schema resourceschema.Schema, model UserModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func userPlan(t *testing.T, schema resourceschema.Schema, model UserModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func userInvalidRaw() tftypes.Value {
	return tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":                                tftypes.String,
			"domain_id":                         tftypes.Number,
			"username":                          tftypes.String,
			"email":                             tftypes.String,
			"first_name":                        tftypes.String,
			"last_name":                         tftypes.String,
			"display_name":                      tftypes.String,
			"force_reset_password":              tftypes.Bool,
			"enabled":                           tftypes.Bool,
			"locked":                            tftypes.Bool,
			"pre_registration":                  tftypes.Bool,
			"reset_password":                    tftypes.String,
			"reset_password_trigger":            tftypes.String,
			"registration_confirmation_trigger": tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":                                tftypes.NewValue(tftypes.String, "user-123"),
			"domain_id":                         tftypes.NewValue(tftypes.Number, 123),
			"username":                          tftypes.NewValue(tftypes.String, "alice"),
			"email":                             tftypes.NewValue(tftypes.String, nil),
			"first_name":                        tftypes.NewValue(tftypes.String, nil),
			"last_name":                         tftypes.NewValue(tftypes.String, nil),
			"display_name":                      tftypes.NewValue(tftypes.String, nil),
			"force_reset_password":              tftypes.NewValue(tftypes.Bool, false),
			"enabled":                           tftypes.NewValue(tftypes.Bool, true),
			"locked":                            tftypes.NewValue(tftypes.Bool, false),
			"pre_registration":                  tftypes.NewValue(tftypes.Bool, true),
			"reset_password":                    tftypes.NewValue(tftypes.String, nil),
			"reset_password_trigger":            tftypes.NewValue(tftypes.String, nil),
			"registration_confirmation_trigger": tftypes.NewValue(tftypes.String, nil),
		},
	)
}
