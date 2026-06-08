package orgidentityprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgIdentityProviderResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"name", "type", "configuration"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"mappers", "domain_whitelist", "external", "group_mapper", "role_mapper"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	for _, name := range []string{"id", "external"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
	if attr := resp.Schema.Attributes["configuration"]; attr == nil || !attr.IsSensitive() {
		t.Fatalf("configuration should be sensitive")
	}
}

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewOrgIdentityProviderResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_identity_provider"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgIdentityProviderResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostics")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgIdentityProviderResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := OrgIdentityProviderModel{
		Name:          types.StringValue("org-idp"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{"users":[]}`),
		External:      types.BoolValue(false),
		DomainWhitelist: []types.String{
			types.StringValue("example.com"),
			types.StringValue("example.org"),
		},
	}

	got := buildCreateBody(plan)
	want := map[string]interface{}{
		"name":            "org-idp",
		"type":            "inline-am-idp",
		"configuration":   `{"users":[]}`,
		"external":        false,
		"domainWhitelist": []string{"example.com", "example.org"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildCreateBodyOmitsAbsentDomainWhitelist(t *testing.T) {
	t.Parallel()

	plan := OrgIdentityProviderModel{
		Name:          types.StringValue("org-idp"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{}`),
		External:      types.BoolValue(false),
	}

	got := buildCreateBody(plan)
	if _, ok := got["domainWhitelist"]; ok {
		t.Fatalf("domainWhitelist should be omitted, got %#v", got)
	}
}

func TestBuildUpdateBodyIncludesOptionalMappings(t *testing.T) {
	t.Parallel()

	groupMapper := mustConditionMap(t, map[string][]string{
		"{#profile['groups'].contains('external-admins')}": {"group-admin"},
		"{#profile['groups'].contains('external-users')}":  {"group-user", "group-admin"},
	})
	roleMapper := mustConditionMap(t, map[string][]string{
		"{#profile['groups'].contains('external-admin-role')}": {"role-admin"},
	})
	plan := OrgIdentityProviderModel{
		Name:          types.StringValue("org-idp"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{"users":[]}`),
		Mappers: map[string]types.String{
			"email":    types.StringValue("email"),
			"username": types.StringValue("username"),
		},
		DomainWhitelist: []types.String{types.StringValue("example.com")},
		GroupMapper:     groupMapper,
		RoleMapper:      roleMapper,
	}

	got := buildUpdateBody(plan, nil)

	if got["name"] != "org-idp" || got["type"] != "inline-am-idp" || got["configuration"] != `{"users":[]}` {
		t.Fatalf("base fields not mapped correctly: %#v", got)
	}
	if !reflect.DeepEqual(got["mappers"], map[string]string{"email": "email", "username": "username"}) {
		t.Fatalf("mappers = %#v", got["mappers"])
	}
	if !reflect.DeepEqual(got["domainWhitelist"], []string{"example.com"}) {
		t.Fatalf("domainWhitelist = %#v", got["domainWhitelist"])
	}

	groupAPI := got["groupMapper"].(map[string][]string)
	assertStringSet(t, groupAPI["group-admin"], []string{
		"{#profile['groups'].contains('external-admins')}",
		"{#profile['groups'].contains('external-users')}",
	})
	assertStringSet(t, groupAPI["group-user"], []string{
		"{#profile['groups'].contains('external-users')}",
	})

	roleAPI := got["roleMapper"].(map[string][]string)
	assertStringSet(t, roleAPI["role-admin"], []string{
		"{#profile['groups'].contains('external-admin-role')}",
	})
}

func TestBuildUpdateBodyClearsRemovedOptionalMappings(t *testing.T) {
	t.Parallel()

	state := OrgIdentityProviderModel{
		Mappers: map[string]types.String{
			"email": types.StringValue("email"),
		},
		DomainWhitelist: []types.String{types.StringValue("example.com")},
		GroupMapper: mustConditionMap(t, map[string][]string{
			"{true}": {"group-id"},
		}),
		RoleMapper: mustConditionMap(t, map[string][]string{
			"{true}": {"role-id"},
		}),
	}
	plan := OrgIdentityProviderModel{
		Name:          types.StringValue("org-idp"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{}`),
		GroupMapper:   types.MapNull(types.ListType{ElemType: types.StringType}),
		RoleMapper:    types.MapNull(types.ListType{ElemType: types.StringType}),
	}

	got := buildUpdateBody(plan, &state)

	if !reflect.DeepEqual(got["mappers"], map[string]string{}) {
		t.Fatalf("mappers clear = %#v", got["mappers"])
	}
	if !reflect.DeepEqual(got["domainWhitelist"], []string{}) {
		t.Fatalf("domainWhitelist clear = %#v", got["domainWhitelist"])
	}
	if !reflect.DeepEqual(got["groupMapper"], map[string][]string{}) {
		t.Fatalf("groupMapper clear = %#v", got["groupMapper"])
	}
	if !reflect.DeepEqual(got["roleMapper"], map[string][]string{}) {
		t.Fatalf("roleMapper clear = %#v", got["roleMapper"])
	}
}

func TestReadIntoModelMapsOrgIdentityProvider(t *testing.T) {
	t.Parallel()

	model := OrgIdentityProviderModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":            "idp-id",
		"name":          "org-idp",
		"type":          "inline-am-idp",
		"external":      true,
		"configuration": map[string]interface{}{"users": []interface{}{}},
		"mappers": map[string]interface{}{
			"email":    "email",
			"username": "username",
		},
		"domainWhitelist": []interface{}{"example.com"},
		"groupMapper": map[string]interface{}{
			"group-id": []interface{}{"{#profile['groups'].contains('external-users')}"},
		},
		"roleMapper": map[string]interface{}{
			"role-id": []interface{}{"{#profile['groups'].contains('external-admin-role')}"},
		},
	})

	if got, want := model.ID.ValueString(), "idp-id"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.Name.ValueString(), "org-idp"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "inline-am-idp"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if !model.External.ValueBool() {
		t.Fatalf("external should be true")
	}
	if got, want := model.Configuration.ValueString(), `{"users":[]}`; got != want {
		t.Fatalf("configuration = %q, want %q", got, want)
	}
	if got, want := model.Mappers["email"].ValueString(), "email"; got != want {
		t.Fatalf("mapper email = %q, want %q", got, want)
	}
	if got, want := stringValues(model.DomainWhitelist), []string{"example.com"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("domainWhitelist = %#v, want %#v", got, want)
	}
	assertListValue(t, model.GroupMapper.Elements()["{#profile['groups'].contains('external-users')}"], []string{"group-id"})
	assertListValue(t, model.RoleMapper.Elements()["{#profile['groups'].contains('external-admin-role')}"], []string{"role-id"})
}

func TestReadIntoModelClearsAbsentOptionalMappings(t *testing.T) {
	t.Parallel()

	model := OrgIdentityProviderModel{
		Mappers:         map[string]types.String{"email": types.StringValue("email")},
		DomainWhitelist: []types.String{types.StringValue("example.com")},
		GroupMapper: mustConditionMap(t, map[string][]string{
			"{true}": {"group-id"},
		}),
		RoleMapper: mustConditionMap(t, map[string][]string{
			"{true}": {"role-id"},
		}),
	}

	readIntoModel(&model, map[string]interface{}{})

	if model.Mappers != nil {
		t.Fatalf("mappers should be nil, got %#v", model.Mappers)
	}
	if model.DomainWhitelist != nil {
		t.Fatalf("domain whitelist should be nil, got %#v", model.DomainWhitelist)
	}
	if !model.GroupMapper.IsNull() {
		t.Fatalf("group mapper should be null")
	}
	if !model.RoleMapper.IsNull() {
		t.Fatalf("role mapper should be null")
	}
}

func TestInvertConditionMapperToAPI(t *testing.T) {
	t.Parallel()

	got := invertConditionMapperToAPI(mustConditionMap(t, map[string][]string{
		"{#profile['groups'].contains('external-admins')}": {"group-admin"},
		"{#profile['groups'].contains('external-users')}":  {"group-user", "group-admin"},
	}))

	assertStringSet(t, got["group-admin"], []string{
		"{#profile['groups'].contains('external-admins')}",
		"{#profile['groups'].contains('external-users')}",
	})
	assertStringSet(t, got["group-user"], []string{
		"{#profile['groups'].contains('external-users')}",
	})
}

func TestReadAPIConditionMapper(t *testing.T) {
	t.Parallel()

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

	assertListValue(t, got.Elements()["{#profile['groups'].contains('external-admins')}"], []string{"group-admin"})
	assertListValue(t, got.Elements()["{#profile['groups'].contains('external-users')}"], []string{"group-user"})
}

func TestOrgIdentityProviderCRUDPreservesMaskedConfigAndConditionMappers(t *testing.T) {
	listType := types.ListType{ElemType: types.StringType}
	groupMapper := mustConditionMap(t, map[string][]string{
		"{#profile['groups'].contains('external-admins')}": {"org-group-admin"},
	})
	roleMapper := mustConditionMap(t, map[string][]string{
		"{#profile['groups'].contains('external-admins')}": {"org-role-admin"},
	})

	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/identities", func(w http.ResponseWriter, r *http.Request) {
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
			"id":            "org-idp-123",
			"name":          body["name"],
			"type":          body["type"],
			"external":      body["external"],
			"configuration": map[string]interface{}{"password": "*****"},
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/identities/org-idp-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":              "org-idp-123",
				"name":            "org-inline",
				"type":            "inline-am-idp",
				"external":        false,
				"configuration":   map[string]interface{}{"password": "*****"},
				"mappers":         map[string]interface{}{"email": "mail"},
				"domainWhitelist": []interface{}{"example.com"},
				"groupMapper": map[string]interface{}{
					"org-group-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
				},
				"roleMapper": map[string]interface{}{
					"org-role-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
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
				"id":              "org-idp-123",
				"name":            body["name"],
				"type":            body["type"],
				"external":        false,
				"configuration":   map[string]interface{}{"password": "*****"},
				"mappers":         body["mappers"],
				"domainWhitelist": body["domainWhitelist"],
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

	resourceUnderTest := &OrgIdentityProviderResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgIdentityProviderPlan(t, schemaResp.Schema, OrgIdentityProviderModel{
		Name:            types.StringValue("org-inline"),
		Type:            types.StringValue("inline-am-idp"),
		External:        types.BoolValue(false),
		Configuration:   types.StringValue(`{"password":"plain"}`),
		Mappers:         map[string]types.String{"email": types.StringValue("mail")},
		DomainWhitelist: []types.String{types.StringValue("example.com")},
		GroupMapper:     groupMapper,
		RoleMapper:      roleMapper,
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState OrgIdentityProviderModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if got := createState.Configuration.ValueString(); got != `{"password":"plain"}` {
		t.Fatalf("configuration after masked create = %q", got)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState OrgIdentityProviderModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got := readState.Configuration.ValueString(); got != `{"password":"plain"}` {
		t.Fatalf("configuration after masked read = %q", got)
	}

	updatePlan := orgIdentityProviderPlan(t, schemaResp.Schema, OrgIdentityProviderModel{
		Name:          types.StringValue("org-inline-updated"),
		Type:          types.StringValue("inline-am-idp"),
		External:      types.BoolValue(false),
		Configuration: types.StringValue(`{"password":"updated"}`),
		GroupMapper:   types.MapNull(listType),
		RoleMapper:    types.MapNull(listType),
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
		"org-group-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
	}) {
		t.Fatalf("post-create groupMapper = %#v", got)
	}
	if got := bodies[1]["roleMapper"]; !reflect.DeepEqual(got, map[string]interface{}{
		"org-role-admin": []interface{}{"{#profile['groups'].contains('external-admins')}"},
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
}

func TestOrgIdentityProviderReadRemovesMissingProviderAndDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/identities/org-idp-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgIdentityProviderResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := orgIdentityProviderState(t, schemaResp.Schema, OrgIdentityProviderModel{
		ID:            types.StringValue("org-idp-123"),
		Name:          types.StringValue("org-inline"),
		Type:          types.StringValue("inline-am-idp"),
		External:      types.BoolValue(false),
		Configuration: types.StringValue(`{"password":"plain"}`),
		GroupMapper:   types.MapNull(types.ListType{ElemType: types.StringType}),
		RoleMapper:    types.MapNull(types.ListType{ElemType: types.StringType}),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatalf("expected missing organization identity provider to remove state, got %#v", readResp.State.Raw)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestOrgIdentityProviderDeleteReportsInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgIdentityProviderResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	listType := tftypes.List{ElementType: tftypes.String}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":               tftypes.Number,
			"name":             tftypes.String,
			"type":             tftypes.String,
			"configuration":    tftypes.String,
			"mappers":          tftypes.Map{ElementType: tftypes.String},
			"domain_whitelist": listType,
			"external":         tftypes.Bool,
			"group_mapper":     tftypes.Map{ElementType: listType},
			"role_mapper":      tftypes.Map{ElementType: listType},
		}},
		map[string]tftypes.Value{
			"id":               tftypes.NewValue(tftypes.Number, 123),
			"name":             tftypes.NewValue(tftypes.String, "org-inline"),
			"type":             tftypes.NewValue(tftypes.String, "inline-am-idp"),
			"configuration":    tftypes.NewValue(tftypes.String, `{"password":"plain"}`),
			"mappers":          tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
			"domain_whitelist": tftypes.NewValue(listType, nil),
			"external":         tftypes.NewValue(tftypes.Bool, false),
			"group_mapper":     tftypes.NewValue(tftypes.Map{ElementType: listType}, nil),
			"role_mapper":      tftypes.NewValue(tftypes.Map{ElementType: listType}, nil),
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

func TestOrgIdentityProviderUpdateReportsInvalidPlanAndStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgIdentityProviderResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	listType := tftypes.List{ElementType: tftypes.String}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":               tftypes.Number,
			"name":             tftypes.String,
			"type":             tftypes.String,
			"configuration":    tftypes.String,
			"mappers":          tftypes.Map{ElementType: tftypes.String},
			"domain_whitelist": listType,
			"external":         tftypes.Bool,
			"group_mapper":     tftypes.Map{ElementType: listType},
			"role_mapper":      tftypes.Map{ElementType: listType},
		}},
		map[string]tftypes.Value{
			"id":               tftypes.NewValue(tftypes.Number, 123),
			"name":             tftypes.NewValue(tftypes.String, "org-inline"),
			"type":             tftypes.NewValue(tftypes.String, "inline-am-idp"),
			"configuration":    tftypes.NewValue(tftypes.String, `{"password":"plain"}`),
			"mappers":          tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
			"domain_whitelist": tftypes.NewValue(listType, nil),
			"external":         tftypes.NewValue(tftypes.Bool, false),
			"group_mapper":     tftypes.NewValue(tftypes.Map{ElementType: listType}, nil),
			"role_mapper":      tftypes.NewValue(tftypes.Map{ElementType: listType}, nil),
		},
	)
	valid := OrgIdentityProviderModel{
		ID:            types.StringValue("org-idp-123"),
		Name:          types.StringValue("org-inline"),
		Type:          types.StringValue("inline-am-idp"),
		Configuration: types.StringValue(`{"password":"plain"}`),
		External:      types.BoolValue(false),
		GroupMapper:   types.MapNull(types.ListType{ElemType: types.StringType}),
		RoleMapper:    types.MapNull(types.ListType{ElemType: types.StringType}),
	}

	invalidPlanResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		State: orgIdentityProviderState(t, schemaResp.Schema, valid),
	}, invalidPlanResp)
	if !invalidPlanResp.Diagnostics.HasError() {
		t.Fatal("expected invalid plan diagnostics")
	}

	invalidStateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  orgIdentityProviderPlan(t, schemaResp.Schema, valid),
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, invalidStateResp)
	if !invalidStateResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestOrgIdentityProviderCRUDReportsRemoteErrors(t *testing.T) {
	tests := map[string]struct {
		createStatus int
		itemStatus   int
		action       func(context.Context, *OrgIdentityProviderResource, tfsdk.Plan, tfsdk.State, resourceschema.Schema) bool
	}{
		"create": {
			createStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *OrgIdentityProviderResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"post_create_update": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *OrgIdentityProviderResource, plan tfsdk.Plan, _ tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"read": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *OrgIdentityProviderResource, _ tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
				r.Read(ctx, resource.ReadRequest{State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"update": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *OrgIdentityProviderResource, plan tfsdk.Plan, state tfsdk.State, schema resourceschema.Schema) bool {
				resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
				return resp.Diagnostics.HasError()
			},
		},
		"delete": {
			itemStatus: http.StatusInternalServerError,
			action: func(ctx context.Context, r *OrgIdentityProviderResource, _ tfsdk.Plan, state tfsdk.State, _ resourceschema.Schema) bool {
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
			mux.HandleFunc("/management/organizations/DEFAULT/identities", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("collection method = %s, want POST", r.Method)
				}
				if tc.createStatus != 0 {
					http.Error(w, "remote error", tc.createStatus)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":            "org-idp-123",
					"name":          "org-inline",
					"type":          "inline-am-idp",
					"external":      false,
					"configuration": map[string]interface{}{"password": "*****"},
				})
			})
			mux.HandleFunc("/management/organizations/DEFAULT/identities/org-idp-123", func(w http.ResponseWriter, r *http.Request) {
				if tc.itemStatus != 0 {
					http.Error(w, "remote error", tc.itemStatus)
					return
				}
				switch r.Method {
				case http.MethodGet, http.MethodPut:
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"id":            "org-idp-123",
						"name":          "org-inline",
						"type":          "inline-am-idp",
						"external":      false,
						"configuration": map[string]interface{}{"password": "*****"},
					})
				case http.MethodDelete:
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Fatalf("item method = %s", r.Method)
				}
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &OrgIdentityProviderResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			planModel := OrgIdentityProviderModel{
				ID:            types.StringValue("org-idp-123"),
				Name:          types.StringValue("org-inline"),
				Type:          types.StringValue("inline-am-idp"),
				External:      types.BoolValue(false),
				Configuration: types.StringValue(`{"password":"plain"}`),
				GroupMapper:   types.MapNull(types.ListType{ElemType: types.StringType}),
				RoleMapper:    types.MapNull(types.ListType{ElemType: types.StringType}),
			}
			if name == "post_create_update" {
				planModel.Mappers = map[string]types.String{"email": types.StringValue("mail")}
			}
			plan := orgIdentityProviderPlan(t, schemaResp.Schema, planModel)
			state := orgIdentityProviderState(t, schemaResp.Schema, planModel)

			if !tc.action(context.Background(), resourceUnderTest, plan, state, schemaResp.Schema) {
				t.Fatal("expected diagnostics")
			}
		})
	}
}

func TestOrgIdentityProviderImportStateSetsID(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	NewOrgIdentityProviderResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := orgIdentityProviderState(t, schemaResp.Schema, OrgIdentityProviderModel{
		ID:            types.StringValue("placeholder"),
		Name:          types.StringValue("org-inline"),
		Type:          types.StringValue("inline-am-idp"),
		External:      types.BoolValue(false),
		Configuration: types.StringValue(`{"password":"plain"}`),
		GroupMapper:   types.MapNull(types.ListType{ElemType: types.StringType}),
		RoleMapper:    types.MapNull(types.ListType{ElemType: types.StringType}),
	})
	importResp := &resource.ImportStateResponse{State: state}

	NewOrgIdentityProviderResource().(resource.ResourceWithImportState).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "org-idp-123",
	}, importResp)

	if importResp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", importResp.Diagnostics)
	}
	var imported OrgIdentityProviderModel
	if diags := importResp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if imported.ID.ValueString() != "org-idp-123" {
		t.Fatalf("id = %q, want org-idp-123", imported.ID.ValueString())
	}
}

func mustConditionMap(t *testing.T, values map[string][]string) types.Map {
	t.Helper()

	listType := types.ListType{ElemType: types.StringType}
	elements := make(map[string]attr.Value, len(values))
	for condition, ids := range values {
		elements[condition] = mustStringList(t, ids...)
	}
	mapVal, diags := types.MapValue(listType, elements)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building mapper: %v", diags)
	}
	return mapVal
}

func orgIdentityProviderPlan(t *testing.T, schema resourceschema.Schema, model OrgIdentityProviderModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func orgIdentityProviderState(t *testing.T, schema resourceschema.Schema, model OrgIdentityProviderModel) tfsdk.State {
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

func stringValues(values []types.String) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.ValueString()
	}
	return result
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
