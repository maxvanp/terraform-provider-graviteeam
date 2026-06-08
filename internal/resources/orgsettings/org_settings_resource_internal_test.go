package orgsettings

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

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewOrgSettingsResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_settings"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgSettingsResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["id"]; attr == nil || !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
	if attr := resp.Schema.Attributes["identities"]; attr == nil || !attr.IsOptional() || !attr.IsComputed() {
		t.Fatalf("identities should be optional+computed")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgSettingsResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgSettingsResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgSettingsResource{}
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

func TestBuildPatchBodyIncludesIdentities(t *testing.T) {
	t.Parallel()

	model := OrgSettingsModel{
		Identities: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("idp-1"),
			types.StringValue("idp-2"),
		}),
	}

	got := buildPatchBody(model)
	want := map[string]interface{}{
		"identities": []string{"idp-1", "idp-2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildPatchBodyAllowsExplicitEmptyIdentities(t *testing.T) {
	t.Parallel()

	model := OrgSettingsModel{
		Identities: types.ListValueMust(types.StringType, []attr.Value{}),
	}

	got := buildPatchBody(model)
	want := map[string]interface{}{
		"identities": []string{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildPatchBodyOmitsUnknownOrNullIdentities(t *testing.T) {
	t.Parallel()

	for name, identities := range map[string]types.List{
		"null":    types.ListNull(types.StringType),
		"unknown": types.ListUnknown(types.StringType),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := buildPatchBody(OrgSettingsModel{Identities: identities})
			if len(got) != 0 {
				t.Fatalf("body = %#v, want empty", got)
			}
		})
	}
}

func TestReadIdentitiesMapsIdentityList(t *testing.T) {
	t.Parallel()

	model := OrgSettingsModel{}

	readIdentities(&model, map[string]interface{}{
		"identities": []interface{}{"idp-1", "idp-2"},
	})

	want := []string{"idp-1", "idp-2"}
	if got := listStrings(t, model.Identities); !reflect.DeepEqual(got, want) {
		t.Fatalf("identities = %#v, want %#v", got, want)
	}
}

func TestReadIdentitiesUsesEmptyListForMissingMalformedOrNilValue(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]interface{}{
		"missing": {},
		"nil": {
			"identities": nil,
		},
		"malformed": {
			"identities": "not-a-list",
		},
	}

	for name, result := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			model := OrgSettingsModel{
				Identities: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("old")}),
			}

			readIdentities(&model, result)

			if got := listStrings(t, model.Identities); len(got) != 0 {
				t.Fatalf("identities = %#v, want empty", got)
			}
		})
	}
}

func TestReadIdentitiesSkipsNonStringEntries(t *testing.T) {
	t.Parallel()

	model := OrgSettingsModel{}

	readIdentities(&model, map[string]interface{}{
		"identities": []interface{}{"idp-1", 42, "idp-2"},
	})

	want := []string{"idp-1", "idp-2"}
	if got := listStrings(t, model.Identities); !reflect.DeepEqual(got, want) {
		t.Fatalf("identities = %#v, want %#v", got, want)
	}
}

func TestOrgSettingsCRUDSkipsEmptyCreatePatchAndDoesNotClearOnDelete(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         "DEFAULT",
				"identities": []interface{}{"idp-existing"},
			})
		case http.MethodPatch:
			methods = append(methods, "patch")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode patch body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         "DEFAULT",
				"identities": body["identities"],
			})
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgSettingsResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgSettingsPlan(t, schemaResp.Schema, OrgSettingsModel{
		Identities: types.ListNull(types.StringType),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}
	var createState OrgSettingsModel
	if diags := createResp.State.Get(context.Background(), &createState); diags.HasError() {
		t.Fatalf("get create state: %#v", diags)
	}
	if got := listStrings(t, createState.Identities); !reflect.DeepEqual(got, []string{"idp-existing"}) {
		t.Fatalf("created identities = %#v", got)
	}

	updatePlan := orgSettingsPlan(t, schemaResp.Schema, OrgSettingsModel{
		Identities: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("idp-1"),
			types.StringValue("idp-2"),
		}),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: createResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}

	deleteResp := &resource.DeleteResponse{State: updateResp.State}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	if !reflect.DeepEqual(methods, []string{"read", "patch", "read"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if len(bodies) != 1 {
		t.Fatalf("patch bodies = %#v, want only update patch", bodies)
	}
	if got := bodies[0]["identities"]; !reflect.DeepEqual(got, []interface{}{"idp-1", "idp-2"}) {
		t.Fatalf("patch identities = %#v", got)
	}
}

func TestOrgSettingsReadSetsSettingsIDAndIdentities(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         "DEFAULT",
			"identities": []interface{}{"idp-1", "idp-2"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgSettingsResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := orgSettingsState(t, schemaResp.Schema, OrgSettingsModel{
		ID: types.StringValue("old"),
		Identities: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("old-idp"),
		}),
	})

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState OrgSettingsModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	if got, want := readState.ID.ValueString(), "settings"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := listStrings(t, readState.Identities), []string{"idp-1", "idp-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("identities = %#v, want %#v", got, want)
	}
}

func TestOrgSettingsCreateReportsPatchError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", r.Method)
		}
		http.Error(w, "patch failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgSettingsResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgSettingsPlan(t, schemaResp.Schema, OrgSettingsModel{
		Identities: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("idp-1")}),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}
}

func TestOrgSettingsCreateReportsReadbackError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "readback failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgSettingsResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgSettingsPlan(t, schemaResp.Schema, OrgSettingsModel{
		Identities: types.ListNull(types.StringType),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected readback diagnostics")
	}
}

func TestOrgSettingsUpdateReportsPatchError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", r.Method)
		}
		http.Error(w, "patch failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgSettingsResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	updatePlan := orgSettingsPlan(t, schemaResp.Schema, OrgSettingsModel{
		Identities: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("idp-2")}),
	})
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &OrgSettingsModel{
		ID:         types.StringValue("settings"),
		Identities: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("idp-1")}),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: updatePlan, State: state}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update patch diagnostics")
	}
}

func TestOrgSettingsReadAndUpdateReportReadErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			http.Error(w, "read failed", http.StatusInternalServerError)
		case http.MethodPatch:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"updated": true})
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgSettingsResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(context.Background(), &OrgSettingsModel{
		ID:         types.StringValue("settings"),
		Identities: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("idp-1")}),
	}); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	updatePlan := orgSettingsPlan(t, schemaResp.Schema, OrgSettingsModel{
		Identities: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("idp-2")}),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: updatePlan, State: state}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}
}

func TestOrgSettingsImportStateSetsSettingsID(t *testing.T) {
	t.Parallel()

	var schemaResp resource.SchemaResponse
	(&OrgSettingsResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: orgSettingsState(t, schemaResp.Schema, OrgSettingsModel{
		ID:         types.StringValue("old"),
		Identities: types.ListValueMust(types.StringType, []attr.Value{}),
	})}

	(&OrgSettingsResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "anything",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state OrgSettingsModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get import state: %#v", diags)
	}
	if got, want := state.ID.ValueString(), "settings"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func orgSettingsPlan(t *testing.T, schema resourceschema.Schema, model OrgSettingsModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func orgSettingsState(t *testing.T, schema resourceschema.Schema, model OrgSettingsModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func listStrings(t *testing.T, list types.List) []string {
	t.Helper()

	values := make([]string, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		value, ok := element.(types.String)
		if !ok {
			t.Fatalf("element %T is not types.String", element)
		}
		values = append(values, value.ValueString())
	}
	return values
}
