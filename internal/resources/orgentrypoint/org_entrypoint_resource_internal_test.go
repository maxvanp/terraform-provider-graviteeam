package orgentrypoint

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
	NewOrgEntrypointResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_entrypoint"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgEntrypointResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"name", "url", "tags"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["description"]; attr == nil || !attr.IsOptional() {
		t.Fatalf("description should be optional")
	}
	for _, name := range []string{"id", "default_entrypoint"} {
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
	(&OrgEntrypointResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgEntrypointResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestBuildBodyForCreate(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{
		Name:        types.StringValue("entrypoint-1"),
		Description: types.StringValue("description"),
		URL:         types.StringValue("https://login.example.com"),
		Tags: []types.String{
			types.StringValue("tag-1"),
			types.StringValue("tag-2"),
		},
	}

	got := buildBody(model)
	want := map[string]interface{}{
		"name":        "entrypoint-1",
		"description": "description",
		"url":         "https://login.example.com",
		"tags":        []string{"tag-1", "tag-2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyClearsRemovedDescriptionForUpdate(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{
		Name:        types.StringValue("entrypoint-1"),
		Description: types.StringNull(),
		URL:         types.StringValue("https://login.example.com"),
		Tags:        []types.String{types.StringValue("tag-1")},
	}
	state := OrgEntrypointModel{
		Description: types.StringValue("old description"),
	}

	got := buildBody(model, state)
	want := map[string]interface{}{
		"name":        "entrypoint-1",
		"description": "",
		"url":         "https://login.example.com",
		"tags":        []string{"tag-1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyOmitsAbsentDescriptionForCreate(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{
		Name:        types.StringValue("entrypoint-1"),
		Description: types.StringNull(),
		URL:         types.StringValue("https://login.example.com"),
		Tags:        []types.String{},
	}

	got := buildBody(model)
	want := map[string]interface{}{
		"name": "entrypoint-1",
		"url":  "https://login.example.com",
		"tags": []string{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelMapsEntrypoint(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":                "entrypoint-id",
		"name":              "entrypoint-1",
		"description":       "description",
		"url":               "https://login.example.com",
		"defaultEntrypoint": true,
		"tags":              []interface{}{"tag-1", "tag-2"},
	})

	if got, want := model.ID.ValueString(), "entrypoint-id"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.Name.ValueString(), "entrypoint-1"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Description.ValueString(), "description"; got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
	if got, want := model.URL.ValueString(), "https://login.example.com"; got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
	if !model.DefaultEntrypoint.ValueBool() {
		t.Fatalf("default entrypoint should be true")
	}
	if got, want := stringValues(model.Tags), []string{"tag-1", "tag-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelClearsEmptyDescriptionAndUnknownDefault(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{
		Description:       types.StringValue("old description"),
		DefaultEntrypoint: types.BoolValue(true),
	}

	readIntoModel(&model, map[string]interface{}{
		"description": "",
	})

	if !model.Description.IsNull() {
		t.Fatalf("description should be null, got %q", model.Description.ValueString())
	}
	if !model.DefaultEntrypoint.IsNull() {
		t.Fatalf("default entrypoint should be null")
	}
}

func TestInterfaceStringsSkipsNonStringEntries(t *testing.T) {
	t.Parallel()

	got := stringValues(interfaceStrings([]interface{}{"tag-1", 42, "tag-2"}))
	want := []string{"tag-1", "tag-2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
}

func TestOrgEntrypointCRUDClearsRemovedDescription(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/entrypoints", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(entrypointResponse("entrypoint-123", body, false))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/entrypoints/entrypoint-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                "entrypoint-123",
				"name":              "entrypoint",
				"description":       "created",
				"url":               "https://login.example.test",
				"tags":              []interface{}{"tag-1"},
				"defaultEntrypoint": true,
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(entrypointResponse("entrypoint-123", body, true))
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgEntrypointResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgEntrypointPlan(t, schemaResp.Schema, OrgEntrypointModel{
		Name:        types.StringValue("entrypoint"),
		Description: types.StringValue("created"),
		URL:         types.StringValue("https://login.example.test"),
		Tags:        []types.String{types.StringValue("tag-1")},
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

	updatePlan := orgEntrypointPlan(t, schemaResp.Schema, OrgEntrypointModel{
		Name:        types.StringValue("entrypoint-updated"),
		Description: types.StringNull(),
		URL:         types.StringValue("https://login-updated.example.test"),
		Tags:        []types.String{types.StringValue("tag-2")},
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
	if got := bodies[1]["description"]; got != "" {
		t.Fatalf("update description = %#v, want clear string", got)
	}
	if got := bodies[1]["tags"]; !reflect.DeepEqual(got, []interface{}{"tag-2"}) {
		t.Fatalf("update tags = %#v", got)
	}
}

func TestOrgEntrypointReadRemovesMissingEntrypointAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing entrypoint", statusCode: http.StatusNotFound, wantRemove: true},
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
			mux.HandleFunc("/management/organizations/DEFAULT/entrypoints/entrypoint-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &OrgEntrypointResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := orgEntrypointState(t, schemaResp.Schema, OrgEntrypointModel{
				ID:   types.StringValue("entrypoint-123"),
				Name: types.StringValue("entrypoint"),
				URL:  types.StringValue("https://login.example.test"),
				Tags: []types.String{types.StringValue("tag-1")},
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing entrypoint to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestOrgEntrypointReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/entrypoints", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/entrypoints/entrypoint-123", func(w http.ResponseWriter, r *http.Request) {
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

	resourceUnderTest := &OrgEntrypointResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := orgEntrypointPlan(t, schemaResp.Schema, OrgEntrypointModel{
		Name: types.StringValue("entrypoint"),
		URL:  types.StringValue("https://login.example.test"),
		Tags: []types.String{types.StringValue("tag-1")},
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := orgEntrypointState(t, schemaResp.Schema, OrgEntrypointModel{
		ID:   types.StringValue("entrypoint-123"),
		Name: types.StringValue("entrypoint"),
		URL:  types.StringValue("https://login.example.test"),
		Tags: []types.String{types.StringValue("tag-1")},
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

func TestOrgEntrypointDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/entrypoints/entrypoint-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgEntrypointResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := orgEntrypointState(t, schemaResp.Schema, OrgEntrypointModel{
		ID:   types.StringValue("entrypoint-123"),
		Name: types.StringValue("entrypoint"),
		URL:  types.StringValue("https://login.example.test"),
		Tags: []types.String{types.StringValue("tag-1")},
	})

	resp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", resp.Diagnostics)
	}
}

func TestOrgEntrypointImportStateSetsID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewOrgEntrypointResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: orgEntrypointState(t, schemaResp.Schema, OrgEntrypointModel{
		ID:   types.StringValue("placeholder"),
		Name: types.StringValue("entrypoint"),
		URL:  types.StringValue("https://login.example.test"),
		Tags: []types.String{types.StringValue("tag-1")},
	})}

	(&OrgEntrypointResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "entrypoint-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported OrgEntrypointModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.ID.ValueString(), "entrypoint-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func TestOrgEntrypointCreateReadUpdateAndDeleteReportInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgEntrypointResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":                 tftypes.String,
			"name":               tftypes.Number,
			"description":        tftypes.String,
			"url":                tftypes.String,
			"tags":               tftypes.List{ElementType: tftypes.String},
			"default_entrypoint": tftypes.Bool,
		}},
		map[string]tftypes.Value{
			"id":                 tftypes.NewValue(tftypes.String, "entrypoint-123"),
			"name":               tftypes.NewValue(tftypes.Number, 123),
			"description":        tftypes.NewValue(tftypes.String, nil),
			"url":                tftypes.NewValue(tftypes.String, "https://login.example.test"),
			"tags":               tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "tag-1")}),
			"default_entrypoint": tftypes.NewValue(tftypes.Bool, false),
		},
	)

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw}}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw}}, readResp)
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

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw}}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func entrypointResponse(id string, body map[string]interface{}, defaultEntrypoint bool) map[string]interface{} {
	return map[string]interface{}{
		"id":                id,
		"name":              body["name"],
		"description":       body["description"],
		"url":               body["url"],
		"tags":              body["tags"],
		"defaultEntrypoint": defaultEntrypoint,
	}
}

func orgEntrypointPlan(t *testing.T, schema resourceschema.Schema, model OrgEntrypointModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func orgEntrypointState(t *testing.T, schema resourceschema.Schema, model OrgEntrypointModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
