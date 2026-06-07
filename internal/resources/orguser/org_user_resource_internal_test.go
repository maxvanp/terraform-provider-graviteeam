package orguser

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
	NewOrgUserResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_user"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgUserResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["username"]; attr == nil || !attr.IsRequired() {
		t.Fatalf("username should be required")
	}
	for _, name := range []string{
		"password",
		"email",
		"first_name",
		"last_name",
		"force_reset_password",
		"enabled",
		"pre_registration",
		"reset_password",
		"reset_password_trigger",
	} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "force_reset_password", "enabled", "pre_registration"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
	for _, name := range []string{"password", "reset_password"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsSensitive() {
			t.Fatalf("attribute %q should be sensitive", name)
		}
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgUserResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildCreateBodyIncludesRequiredAndOptionalFields(t *testing.T) {
	t.Parallel()

	model := OrgUserModel{
		Username:           types.StringValue("alice"),
		Password:           types.StringValue("secret"),
		Email:              types.StringValue("alice@example.com"),
		FirstName:          types.StringValue("Alice"),
		LastName:           types.StringValue("Liddell"),
		ForceResetPassword: types.BoolValue(true),
		PreRegistration:    types.BoolValue(false),
	}

	got := buildCreateBody(model)
	want := map[string]interface{}{
		"username":           "alice",
		"password":           "secret",
		"email":              "alice@example.com",
		"firstName":          "Alice",
		"lastName":           "Liddell",
		"forceResetPassword": true,
		"preRegistration":    false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("create body = %#v, want %#v", got, want)
	}
}

func TestBuildCreateBodyOmitsUnknownPasswordAndNullProfileFields(t *testing.T) {
	t.Parallel()

	model := OrgUserModel{
		Username:           types.StringValue("alice"),
		Password:           types.StringUnknown(),
		Email:              types.StringNull(),
		FirstName:          types.StringNull(),
		LastName:           types.StringNull(),
		ForceResetPassword: types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
	}

	got := buildCreateBody(model)
	want := map[string]interface{}{
		"username":           "alice",
		"forceResetPassword": false,
		"preRegistration":    true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("create body = %#v, want %#v", got, want)
	}
}

func TestBuildMergedUpdateBodyPreservesUnmanagedCurrentFields(t *testing.T) {
	t.Parallel()

	current := map[string]interface{}{
		"id":                    "ignored",
		"accountNonExpired":     true,
		"accountNonLocked":      true,
		"additionalInformation": map[string]interface{}{"department": "platform"},
		"client":                "api",
		"createdAt":             "2026-01-01T00:00:00Z",
		"credentialsNonExpired": true,
		"displayName":           "Old Name",
		"email":                 "old@example.com",
		"enabled":               true,
		"externalId":            "external-1",
		"firstName":             "Old",
		"forceResetPassword":    false,
		"lastName":              "Name",
		"loggedAt":              "2026-01-02T00:00:00Z",
		"loginsCount":           float64(3),
		"preRegistration":       true,
		"preferredLanguage":     "fr",
		"registrationCompleted": false,
		"source":                "gravitee",
		"updatedAt":             "2026-01-03T00:00:00Z",
		"unexpected":            "must-not-leak",
	}
	plan := OrgUserModel{
		Email:              types.StringValue("new@example.com"),
		FirstName:          types.StringValue("New"),
		LastName:           types.StringValue("Person"),
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
		"displayName",
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
	if got["email"] != "new@example.com" || got["firstName"] != "New" || got["lastName"] != "Person" {
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
		"email":     "old@example.com",
		"firstName": "Old",
		"lastName":  "Name",
	}
	plan := OrgUserModel{
		Email:              types.StringNull(),
		FirstName:          types.StringUnknown(),
		LastName:           types.StringNull(),
		ForceResetPassword: types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
	}

	got := buildMergedUpdateBody(current, plan)

	if got["email"] != current["email"] || got["firstName"] != current["firstName"] || got["lastName"] != current["lastName"] {
		t.Fatalf("current optional fields were not preserved: %#v", got)
	}
}

func TestShouldResetPassword(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		plan types.String
		prev types.String
		want bool
	}{
		"null plan does not reset":         {types.StringNull(), types.StringValue("old"), false},
		"unknown plan does not reset":      {types.StringUnknown(), types.StringValue("old"), false},
		"new trigger resets":               {types.StringValue("new"), types.StringNull(), true},
		"changed trigger resets":           {types.StringValue("new"), types.StringValue("old"), true},
		"unchanged trigger does not reset": {types.StringValue("same"), types.StringValue("same"), false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := shouldResetPassword(test.plan, test.prev); got != test.want {
				t.Fatalf("shouldResetPassword = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReadIntoModelMapsAPIFieldsAndClearsMissingOptionalStrings(t *testing.T) {
	t.Parallel()

	model := OrgUserModel{
		Email:     types.StringValue("previous@example.com"),
		FirstName: types.StringValue("Previous"),
		LastName:  types.StringValue("Name"),
	}
	readIntoModel(&model, map[string]interface{}{
		"id":                 "user-1",
		"username":           "alice",
		"forceResetPassword": true,
		"enabled":            false,
		"preRegistration":    true,
	})

	if model.ID.ValueString() != "user-1" || model.Username.ValueString() != "alice" {
		t.Fatalf("identity fields not mapped: %#v", model)
	}
	if !model.Email.IsNull() || !model.FirstName.IsNull() || !model.LastName.IsNull() {
		t.Fatalf("optional strings were not cleared: %#v", model)
	}
	if !model.ForceResetPassword.ValueBool() || model.Enabled.ValueBool() || !model.PreRegistration.ValueBool() {
		t.Fatalf("boolean fields not mapped: %#v", model)
	}
}

func TestOrgUserCRUDUsesMergedProfileUpdateAndSeparateActions(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(orgUserResponse("org-user-123", map[string]interface{}{
			"username":           body["username"],
			"email":              body["email"],
			"firstName":          body["firstName"],
			"lastName":           body["lastName"],
			"forceResetPassword": body["forceResetPassword"],
			"preRegistration":    body["preRegistration"],
			"enabled":            true,
		}))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/org-user-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(orgUserResponse("org-user-123", map[string]interface{}{
				"username":              "alice",
				"email":                 "alice@example.com",
				"firstName":             "Alice",
				"lastName":              "Liddell",
				"forceResetPassword":    false,
				"preRegistration":       true,
				"enabled":               false,
				"accountNonExpired":     true,
				"accountNonLocked":      true,
				"additionalInformation": map[string]interface{}{"department": "platform"},
				"credentialsNonExpired": true,
				"displayName":           "Alice Liddell",
				"externalId":            "external-1",
				"preferredLanguage":     "fr",
				"registrationCompleted": false,
				"source":                "gravitee",
			}))
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(orgUserResponse("org-user-123", body))
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/org-user-123/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("status method = %s, want PUT", r.Method)
		}
		methods = append(methods, "status")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode status body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(orgUserResponse("org-user-123", map[string]interface{}{
			"username":           "alice",
			"email":              "alice@example.com",
			"firstName":          "Alice",
			"lastName":           "Liddell",
			"forceResetPassword": false,
			"preRegistration":    true,
			"enabled":            body["enabled"],
		}))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/org-user-123/username", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("username method = %s, want PATCH", r.Method)
		}
		methods = append(methods, "username")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode username body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(orgUserResponse("org-user-123", map[string]interface{}{
			"username": body["username"],
		}))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/users/org-user-123/resetPassword", func(w http.ResponseWriter, r *http.Request) {
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
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgUserResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgUserPlan(t, schemaResp.Schema, OrgUserModel{
		Username:           types.StringValue("alice"),
		Password:           types.StringValue("initial-secret"),
		Email:              types.StringValue("alice@example.com"),
		FirstName:          types.StringValue("Alice"),
		LastName:           types.StringValue("Liddell"),
		ForceResetPassword: types.BoolValue(false),
		Enabled:            types.BoolValue(false),
		PreRegistration:    types.BoolValue(true),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState OrgUserModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if createState.Password.ValueString() != "initial-secret" {
		t.Fatalf("password after create = %q", createState.Password.ValueString())
	}
	if createState.Enabled.ValueBool() {
		t.Fatalf("enabled should be false after post-create status update")
	}

	updatePlan := orgUserPlan(t, schemaResp.Schema, OrgUserModel{
		Username:           types.StringValue("alice-updated"),
		Password:           types.StringValue("initial-secret"),
		Email:              types.StringValue("alice-updated@example.com"),
		FirstName:          types.StringValue("Alice"),
		LastName:           types.StringValue("Updated"),
		ForceResetPassword: types.BoolValue(true),
		Enabled:            types.BoolValue(true),
		PreRegistration:    types.BoolValue(false),
		ResetPassword:      types.StringValue("rotated-secret"),
		ResetTrigger:       types.StringValue("rotation-1"),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: createResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}
	var updateState OrgUserModel
	if diags := updateResp.State.Get(context.Background(), &updateState); diags.HasError() {
		t.Fatalf("get update state: %#v", diags)
	}
	if updateState.Password.ValueString() != "initial-secret" {
		t.Fatalf("password after update = %q", updateState.Password.ValueString())
	}
	if updateState.ResetPassword.ValueString() != "rotated-secret" {
		t.Fatalf("reset_password after update = %q", updateState.ResetPassword.ValueString())
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if !reflect.DeepEqual(methods, []string{"create", "status", "username", "read", "update", "status", "reset", "read", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if got := bodies[0]["password"]; got != "initial-secret" {
		t.Fatalf("create password = %#v", got)
	}
	if got := bodies[1]["enabled"]; got != false {
		t.Fatalf("create status body = %#v", bodies[1])
	}
	if got := bodies[2]["username"]; got != "alice-updated" {
		t.Fatalf("username body = %#v", bodies[2])
	}
	updateBody := bodies[3]
	for _, field := range []string{"accountNonExpired", "accountNonLocked", "additionalInformation", "credentialsNonExpired", "displayName", "externalId", "preferredLanguage", "registrationCompleted", "source"} {
		if _, ok := updateBody[field]; !ok {
			t.Fatalf("merged update body missing preserved field %q: %#v", field, updateBody)
		}
	}
	if updateBody["email"] != "alice-updated@example.com" || updateBody["lastName"] != "Updated" {
		t.Fatalf("merged update body did not apply plan fields: %#v", updateBody)
	}
	if updateBody["forceResetPassword"] != true || updateBody["preRegistration"] != false {
		t.Fatalf("merged update body did not apply plan booleans: %#v", updateBody)
	}
	if got := bodies[4]["enabled"]; got != true {
		t.Fatalf("update status body = %#v", bodies[4])
	}
	if got := bodies[5]["password"]; got != "rotated-secret" {
		t.Fatalf("reset password body = %#v", bodies[5])
	}
}

func orgUserResponse(id string, body map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"id": id,
	}
	for key, value := range body {
		result[key] = value
	}
	return result
}

func orgUserPlan(t *testing.T, schema resourceschema.Schema, model OrgUserModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
