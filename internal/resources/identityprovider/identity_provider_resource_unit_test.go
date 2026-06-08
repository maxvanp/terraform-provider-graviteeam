package identityprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewIdentityProviderResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_identity_provider"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewIdentityProviderResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", true, false, false)
	assertBoolAttribute(t, resp.Schema.Attributes, "external", false, true, true)
	assertStringAttribute(t, resp.Schema.Attributes, "configuration", true, false, false)
	assertMapAttribute(t, resp.Schema.Attributes, "mappers", types.StringType)
	assertListAttribute(t, resp.Schema.Attributes, "domain_whitelist", types.StringType)
	assertStringAttribute(t, resp.Schema.Attributes, "password_policy_id", false, true, false)
	assertMapAttribute(t, resp.Schema.Attributes, "group_mapper", types.ListType{ElemType: types.StringType})
	assertMapAttribute(t, resp.Schema.Attributes, "role_mapper", types.ListType{ElemType: types.StringType})
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&IdentityProviderResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&IdentityProviderResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &IdentityProviderResource{}
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

func TestInvertConditionMapperToAPI(t *testing.T) {
	listType := types.ListType{ElemType: types.StringType}
	mapper, diags := types.MapValue(listType, map[string]attr.Value{
		"{#profile['groups'].contains('external-admins')}": mustStringList(t, "group-admin"),
		"{#profile['groups'].contains('external-users')}":  mustStringList(t, "group-user", "group-admin"),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building mapper: %v", diags)
	}

	got := invertConditionMapperToAPI(mapper)

	assertStringSet(t, got["group-admin"], []string{
		"{#profile['groups'].contains('external-admins')}",
		"{#profile['groups'].contains('external-users')}",
	})
	assertStringSet(t, got["group-user"], []string{
		"{#profile['groups'].contains('external-users')}",
	})
}

func TestReadAPIConditionMapper(t *testing.T) {
	listType := types.ListType{ElemType: types.StringType}
	got := readAPIConditionMapper(map[string]interface{}{
		"groupMapper": map[string]interface{}{
			"group-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
			"group-user":  []interface{}{"{#profile['groups'].contains('external-users')}"},
		},
	}, "groupMapper", listType)

	if got.IsNull() || got.IsUnknown() {
		t.Fatalf("expected concrete mapper, got %v", got)
	}

	elements := got.Elements()
	if len(elements) != 2 {
		t.Fatalf("expected 2 mapper entries, got %d", len(elements))
	}
	assertListValue(t, elements["{#profile['groups'].contains('external-admins')}"], []string{"group-admin"})
	assertListValue(t, elements["{#profile['groups'].contains('external-users')}"], []string{"group-user"})
}

func TestBuildUpdateBodyClearsRemovedIdentityProviderCollections(t *testing.T) {
	t.Parallel()

	listType := types.ListType{ElemType: types.StringType}
	stateMapper, diags := types.MapValue(listType, map[string]attr.Value{
		"{#profile['groups'].contains('legacy')}": mustStringList(t, "group-old"),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building mapper: %v", diags)
	}

	plan := IdentityProviderModel{
		Name:          types.StringValue("inline"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{"users":[]}`),
		GroupMapper:   types.MapNull(listType),
		RoleMapper:    types.MapNull(listType),
	}
	state := IdentityProviderModel{
		Mappers:         map[string]types.String{"email": types.StringValue("mail")},
		DomainWhitelist: []types.String{types.StringValue("example.com")},
		GroupMapper:     stateMapper,
		RoleMapper:      stateMapper,
	}

	got := (&IdentityProviderResource{}).buildUpdateBody(plan, &state)

	if !reflect.DeepEqual(got["mappers"], map[string]string{}) {
		t.Fatalf("mappers = %#v, want empty map", got["mappers"])
	}
	if !reflect.DeepEqual(got["domainWhitelist"], []string{}) {
		t.Fatalf("domainWhitelist = %#v, want empty list", got["domainWhitelist"])
	}
	if !reflect.DeepEqual(got["groupMapper"], map[string][]string{}) {
		t.Fatalf("groupMapper = %#v, want empty map", got["groupMapper"])
	}
	if !reflect.DeepEqual(got["roleMapper"], map[string][]string{}) {
		t.Fatalf("roleMapper = %#v, want empty map", got["roleMapper"])
	}
}

func TestIdentityProviderCRUDPreservesMaskedConfigAndConditionMappers(t *testing.T) {
	listType := types.ListType{ElemType: types.StringType}
	groupMapper, diags := types.MapValue(listType, map[string]attr.Value{
		"{#profile['groups'].contains('external-admins')}": mustStringList(t, "group-admin"),
	})
	if diags.HasError() {
		t.Fatalf("build group mapper: %#v", diags)
	}
	roleMapper, diags := types.MapValue(listType, map[string]attr.Value{
		"{#profile['groups'].contains('external-admins')}": mustStringList(t, "role-admin"),
	})
	if diags.HasError() {
		t.Fatalf("build role mapper: %#v", diags)
	}

	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities", func(w http.ResponseWriter, r *http.Request) {
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
			"id":            "idp-123",
			"name":          body["name"],
			"type":          body["type"],
			"external":      body["external"],
			"configuration": map[string]interface{}{"password": "*****"},
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities/idp-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":              "idp-123",
				"name":            "inline",
				"type":            "inline-am-idp",
				"external":        false,
				"configuration":   map[string]interface{}{"password": "*****"},
				"mappers":         map[string]interface{}{"email": "mail"},
				"domainWhitelist": []interface{}{"example.com"},
				"groupMapper": map[string]interface{}{
					"group-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
				},
				"roleMapper": map[string]interface{}{
					"role-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
				},
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":              "idp-123",
				"name":            body["name"],
				"type":            body["type"],
				"external":        false,
				"configuration":   map[string]interface{}{"password": "*****"},
				"mappers":         body["mappers"],
				"domainWhitelist": body["domainWhitelist"],
				"passwordPolicy":  body["passwordPolicy"],
				"groupMapper":     body["groupMapper"],
				"roleMapper":      body["roleMapper"],
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

	resourceUnderTest := &IdentityProviderResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := identityProviderPlan(t, schemaResp.Schema, IdentityProviderModel{
		DomainID:         types.StringValue("domain-123"),
		Name:             types.StringValue("inline"),
		Type:             types.StringValue("inline-am-idp"),
		External:         types.BoolValue(false),
		Configuration:    types.StringValue(`{"password":"plain"}`),
		Mappers:          map[string]types.String{"email": types.StringValue("mail")},
		DomainWhitelist:  []types.String{types.StringValue("example.com")},
		PasswordPolicyID: types.StringValue("password-policy-1"),
		GroupMapper:      groupMapper,
		RoleMapper:       roleMapper,
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState IdentityProviderModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if got := createState.Configuration.ValueString(); got != `{"password":"plain"}` {
		t.Fatalf("configuration after masked create = %q", got)
	}
	if got := createState.PasswordPolicyID.ValueString(); got != "password-policy-1" {
		t.Fatalf("password policy after create = %q", got)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState IdentityProviderModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got := readState.Configuration.ValueString(); got != `{"password":"plain"}` {
		t.Fatalf("configuration after masked read = %q", got)
	}

	updatePlan := identityProviderPlan(t, schemaResp.Schema, IdentityProviderModel{
		DomainID:         types.StringValue("domain-123"),
		Name:             types.StringValue("inline-updated"),
		Type:             types.StringValue("inline-am-idp"),
		External:         types.BoolValue(false),
		Configuration:    types.StringValue(`{"password":"updated"}`),
		PasswordPolicyID: types.StringNull(),
		GroupMapper:      types.MapNull(listType),
		RoleMapper:       types.MapNull(listType),
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

	if !reflect.DeepEqual(methods, []string{"create", "update", "read", "update", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if len(bodies) != 3 {
		t.Fatalf("bodies = %#v, want create, post-create update, update", bodies)
	}
	if _, ok := bodies[0]["mappers"]; ok {
		t.Fatalf("create body should not include mappers: %#v", bodies[0])
	}
	if got := bodies[1]["groupMapper"]; !reflect.DeepEqual(got, map[string]interface{}{
		"group-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
	}) {
		t.Fatalf("post-create groupMapper = %#v", got)
	}
	if got := bodies[1]["roleMapper"]; !reflect.DeepEqual(got, map[string]interface{}{
		"role-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
	}) {
		t.Fatalf("post-create roleMapper = %#v", got)
	}
	if got := bodies[2]["configuration"]; got != `{"password":"updated"}` {
		t.Fatalf("update configuration = %#v", got)
	}
	if got := bodies[2]["mappers"]; !reflect.DeepEqual(got, map[string]interface{}{}) {
		t.Fatalf("update mappers = %#v, want clear map", got)
	}
	if got := bodies[2]["domainWhitelist"]; !reflect.DeepEqual(got, []interface{}{}) {
		t.Fatalf("update domainWhitelist = %#v, want clear list", got)
	}
	if got := bodies[2]["groupMapper"]; !reflect.DeepEqual(got, map[string]interface{}{}) {
		t.Fatalf("update groupMapper = %#v, want clear map", got)
	}
	if got := bodies[2]["roleMapper"]; !reflect.DeepEqual(got, map[string]interface{}{}) {
		t.Fatalf("update roleMapper = %#v, want clear map", got)
	}
	if _, ok := bodies[2]["passwordPolicy"]; ok {
		t.Fatalf("update body should not clear passwordPolicy implicitly: %#v", bodies[2])
	}
}

func TestIdentityProviderReadRemovesMissingProviderAndDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities/idp-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &IdentityProviderResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	listType := types.ListType{ElemType: types.StringType}
	state := identityProviderState(t, schemaResp.Schema, IdentityProviderModel{
		ID:            types.StringValue("idp-123"),
		DomainID:      types.StringValue("domain-123"),
		Name:          types.StringValue("inline"),
		Type:          types.StringValue("inline-am-idp"),
		External:      types.BoolValue(false),
		Configuration: types.StringValue(`{"users":[]}`),
		GroupMapper:   types.MapNull(listType),
		RoleMapper:    types.MapNull(listType),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing identity provider to remove state, got %#v", readResp.State.Raw)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestIdentityProviderDeleteReportsInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &IdentityProviderResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	listType := tftypes.List{ElementType: tftypes.String}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":                 tftypes.String,
			"domain_id":          tftypes.Number,
			"name":               tftypes.String,
			"type":               tftypes.String,
			"external":           tftypes.Bool,
			"configuration":      tftypes.String,
			"mappers":            tftypes.Map{ElementType: tftypes.String},
			"domain_whitelist":   listType,
			"password_policy_id": tftypes.String,
			"group_mapper":       tftypes.Map{ElementType: listType},
			"role_mapper":        tftypes.Map{ElementType: listType},
		}},
		map[string]tftypes.Value{
			"id":                 tftypes.NewValue(tftypes.String, "idp-123"),
			"domain_id":          tftypes.NewValue(tftypes.Number, 123),
			"name":               tftypes.NewValue(tftypes.String, "inline"),
			"type":               tftypes.NewValue(tftypes.String, "inline-am-idp"),
			"external":           tftypes.NewValue(tftypes.Bool, false),
			"configuration":      tftypes.NewValue(tftypes.String, `{"users":[]}`),
			"mappers":            tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
			"domain_whitelist":   tftypes.NewValue(listType, nil),
			"password_policy_id": tftypes.NewValue(tftypes.String, nil),
			"group_mapper":       tftypes.NewValue(tftypes.Map{ElementType: listType}, nil),
			"role_mapper":        tftypes.NewValue(tftypes.Map{ElementType: listType}, nil),
		},
	)

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestIdentityProviderCRUDReportsRemoteErrors(t *testing.T) {
	tests := map[string]struct {
		createStatus int
		itemStatus   int
		action       func(context.Context, *IdentityProviderResource, tfsdk.Plan, tfsdk.State, resourceschema.Schema) bool
	}{
		"create": {
			createStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *IdentityProviderResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"post_create_update": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *IdentityProviderResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"read": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *IdentityProviderResource, _ tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
				r.Read(ctx, resource.ReadRequest{State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"update": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *IdentityProviderResource, plan tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"delete": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *IdentityProviderResource, _ tfsdk.Plan, state tfsdk.State, _ resourceschema.Schema) bool {
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
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("collection method = %s, want POST", r.Method)
				}
				if tc.createStatus != 0 {
					http.Error(w, "remote error", tc.createStatus)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":            "idp-123",
					"name":          "inline",
					"type":          "inline-am-idp",
					"external":      false,
					"configuration": map[string]interface{}{"users": []interface{}{}},
				})
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/identities/idp-123", func(w http.ResponseWriter, r *http.Request) {
				if tc.itemStatus != 0 {
					http.Error(w, "remote error", tc.itemStatus)
					return
				}
				switch r.Method {
				case http.MethodGet, http.MethodPut:
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"id":            "idp-123",
						"name":          "inline",
						"type":          "inline-am-idp",
						"external":      false,
						"configuration": map[string]interface{}{"users": []interface{}{}},
					})
				case http.MethodDelete:
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Fatalf("item method = %s", r.Method)
				}
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &IdentityProviderResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			listType := types.ListType{ElemType: types.StringType}
			planModel := IdentityProviderModel{
				ID:            types.StringValue("idp-123"),
				DomainID:      types.StringValue("domain-123"),
				Name:          types.StringValue("inline"),
				Type:          types.StringValue("inline-am-idp"),
				External:      types.BoolValue(false),
				Configuration: types.StringValue(`{"users":[]}`),
				GroupMapper:   types.MapNull(listType),
				RoleMapper:    types.MapNull(listType),
			}
			if name == "post_create_update" {
				planModel.Mappers = map[string]types.String{"email": types.StringValue("mail")}
			}
			plan := identityProviderPlan(t, schemaResp.Schema, planModel)
			state := identityProviderState(t, schemaResp.Schema, planModel)

			if !tc.action(context.Background(), resourceUnderTest, plan, state, schemaResp.Schema) {
				t.Fatal("expected diagnostics")
			}
		})
	}
}

func TestIdentityProviderImportStateRejectsInvalidID(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewIdentityProviderResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	importResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	NewIdentityProviderResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "idp-only",
	}, importResp)

	if !importResp.Diagnostics.HasError() {
		t.Fatal("expected invalid import diagnostics")
	}
}

func TestIdentityProviderImportStateSetsDomainAndID(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewIdentityProviderResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	listType := types.ListType{ElemType: types.StringType}
	importResp := &resource.ImportStateResponse{State: identityProviderState(t, schemaResp.Schema, IdentityProviderModel{
		ID:            types.StringValue("placeholder"),
		DomainID:      types.StringValue("placeholder"),
		Name:          types.StringValue("inline"),
		Type:          types.StringValue("inline-am-idp"),
		External:      types.BoolValue(false),
		Configuration: types.StringValue(`{"users":[]}`),
		GroupMapper:   types.MapNull(listType),
		RoleMapper:    types.MapNull(listType),
	})}

	NewIdentityProviderResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/idp-123",
	}, importResp)

	if importResp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", importResp.Diagnostics)
	}
	var imported IdentityProviderModel
	if diags := importResp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.ID.ValueString(), "idp-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func identityProviderPlan(t *testing.T, schema resourceschema.Schema, model IdentityProviderModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func identityProviderState(t *testing.T, schema resourceschema.Schema, model IdentityProviderModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func mustStringList(t *testing.T, values ...string) types.List {
	t.Helper()

	listValues := make([]attr.Value, len(values))
	for i, value := range values {
		listValues[i] = types.StringValue(value)
	}
	list, diags := types.ListValue(types.StringType, listValues)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building list: %v", diags)
	}
	return list
}

func assertStringSet(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	seen := make(map[string]bool, len(got))
	for _, value := range got {
		seen[value] = true
	}
	for _, value := range want {
		if !seen[value] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func assertListValue(t *testing.T, got attr.Value, want []string) {
	t.Helper()

	list, ok := got.(types.List)
	if !ok {
		t.Fatalf("expected types.List, got %T", got)
	}
	values := make([]string, 0, len(list.Elements()))
	for _, elem := range list.Elements() {
		value, ok := elem.(types.String)
		if !ok {
			t.Fatalf("expected types.String, got %T", elem)
		}
		values = append(values, value.ValueString())
	}
	assertStringSet(t, values, want)
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

func assertMapAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, elemType attr.Type) {
	t.Helper()

	attr, ok := attrs[name].(schema.MapAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.MapAttribute", name, attrs[name])
	}
	if !attr.Optional || attr.Required || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want optional only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
	if !reflect.DeepEqual(attr.ElementType, elemType) {
		t.Fatalf("%s element type = %#v, want %#v", name, attr.ElementType, elemType)
	}
}

func assertListAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, elemType attr.Type) {
	t.Helper()

	attr, ok := attrs[name].(schema.ListAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.ListAttribute", name, attrs[name])
	}
	if !attr.Optional || attr.Required || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want optional only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
	if !reflect.DeepEqual(attr.ElementType, elemType) {
		t.Fatalf("%s element type = %#v, want %#v", name, attr.ElementType, elemType)
	}
}
