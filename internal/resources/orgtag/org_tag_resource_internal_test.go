package orgtag

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

func TestOrgTagMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewOrgTagResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_org_tag" {
		t.Fatalf("type name = %q, want graviteeam_org_tag", resp.TypeName)
	}
}

func TestOrgTagSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgTagResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["name"]; !attr.IsRequired() {
		t.Fatal("name should be required")
	}
	if attr := resp.Schema.Attributes["description"]; !attr.IsOptional() {
		t.Fatal("description should be optional")
	}
	if attr := resp.Schema.Attributes["id"]; !attr.IsComputed() {
		t.Fatal("id should be computed")
	}
}

func TestOrgTagConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &OrgTagResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildBodyForCreate(t *testing.T) {
	t.Parallel()

	plan := OrgTagModel{
		Name:        types.StringValue("tag-1"),
		Description: types.StringValue("description"),
	}

	got := buildBody(plan, nil)
	want := map[string]interface{}{
		"name":        "tag-1",
		"description": "description",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyOmitsAbsentDescriptionForCreate(t *testing.T) {
	t.Parallel()

	plan := OrgTagModel{
		Name:        types.StringValue("tag-1"),
		Description: types.StringNull(),
	}

	got := buildBody(plan, nil)
	want := map[string]interface{}{
		"name": "tag-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyClearsRemovedDescriptionForUpdate(t *testing.T) {
	t.Parallel()

	plan := OrgTagModel{
		Name:        types.StringValue("tag-1"),
		Description: types.StringNull(),
	}
	state := OrgTagModel{
		Name:        types.StringValue("tag-1"),
		Description: types.StringValue("old description"),
	}

	got := buildBody(plan, &state)
	want := map[string]interface{}{
		"name":        "tag-1",
		"description": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelMapsTag(t *testing.T) {
	t.Parallel()

	model := OrgTagModel{}

	readIntoModel(&model, map[string]interface{}{
		"name":        "tag-1",
		"description": "description",
	})

	if got, want := model.Name.ValueString(), "tag-1"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Description.ValueString(), "description"; got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
}

func TestReadIntoModelClearsEmptyDescription(t *testing.T) {
	t.Parallel()

	model := OrgTagModel{
		Description: types.StringValue("old description"),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":        "tag-1",
		"description": "",
	})

	if got, want := model.Name.ValueString(), "tag-1"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if !model.Description.IsNull() {
		t.Fatalf("description should be null, got %q", model.Description.ValueString())
	}
}

func TestOrgTagCRUDClearsRemovedDescription(t *testing.T) {
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
	var deletePaths []string
	mux.HandleFunc("/management/organizations/DEFAULT/tags", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected tag collection method %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "tag-123",
			"name":        body["name"],
			"description": body["description"],
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/tags/tag-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "tag-123",
				"name":        "shard-a",
				"description": "from api",
			})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "tag-123",
				"name":        body["name"],
				"description": body["description"],
			})
		case http.MethodDelete:
			deletePaths = append(deletePaths, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected tag item method %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &OrgTagResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := orgTagPlan(t, schemaResp.Schema, OrgTagModel{
		Name:        types.StringValue("shard-a"),
		Description: types.StringValue("created"),
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

	updatePlan := orgTagPlan(t, schemaResp.Schema, OrgTagModel{
		Name:        types.StringValue("shard-a"),
		Description: types.StringNull(),
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

	wantBodies := []map[string]interface{}{
		{"name": "shard-a", "description": "created"},
		{"name": "shard-a", "description": ""},
	}
	if !reflect.DeepEqual(bodies, wantBodies) {
		t.Fatalf("bodies = %#v, want %#v", bodies, wantBodies)
	}
	if want := []string{"/management/organizations/DEFAULT/tags/tag-123"}; !reflect.DeepEqual(deletePaths, want) {
		t.Fatalf("delete paths = %#v, want %#v", deletePaths, want)
	}
}

func orgTagPlan(t *testing.T, schema resourceschema.Schema, model OrgTagModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}
