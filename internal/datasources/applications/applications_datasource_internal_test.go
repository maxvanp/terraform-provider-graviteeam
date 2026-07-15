package applications

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadataSchemaAndConfigure(t *testing.T) {
	t.Parallel()

	dataSource := &ApplicationsDataSource{}
	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "graviteeam"}, &metadataResp)
	if metadataResp.TypeName != "graviteeam_applications" {
		t.Fatalf("type name = %q", metadataResp.TypeName)
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if attr := schemaResp.Schema.Attributes["domain_id"]; attr == nil || !attr.IsRequired() {
		t.Fatal("domain_id should be required")
	}
	if attr := schemaResp.Schema.Attributes["result_json"]; attr == nil || !attr.IsComputed() {
		t.Fatal("result_json should be computed")
	}

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &datasource.ConfigureResponse{})
	if dataSource.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestBuildQueryCoversFiltersAndValidation(t *testing.T) {
	t.Parallel()

	model := ApplicationsModel{
		PaginationMode: types.StringValue("cursor"),
		Limit:          types.Int64Value(25),
		Sort:           types.StringValue("name"),
		Direction:      types.StringValue("ASC"),
		Page:           types.Int64Value(2),
		Query:          types.StringValue("agent"),
		Status:         types.StringValue("enabled"),
		OwnerEmail:     types.StringValue("owner@example.com"),
		Cursor:         types.StringValue("next-token"),
		Expand:         []types.String{types.StringValue("owners")},
		ApplicationTypes: []types.String{
			types.StringValue("WEB"), types.StringValue("AGENT"),
		},
	}
	query, cursorMode, err := buildQuery(model)
	if err != nil {
		t.Fatal(err)
	}
	if !cursorMode {
		t.Fatal("expected cursor mode")
	}
	wantTypes := []string{"WEB", "AGENT"}
	if !reflect.DeepEqual(query["type"], wantTypes) || query.Get("owner.email") != "owner@example.com" || query.Get("cursor") != "next-token" {
		t.Fatalf("query = %#v", query)
	}

	model.PaginationMode = types.StringValue("page")
	if _, _, err := buildQuery(model); err == nil {
		t.Fatal("expected cursor in page mode to fail")
	}
	model.Cursor = types.StringNull()
	model.ApplicationTypes = []types.String{types.StringValue("INVALID")}
	if _, _, err := buildQuery(model); err == nil {
		t.Fatal("expected invalid application type to fail")
	}
}

func TestSearchApplicationsUsesBothEndpoints(t *testing.T) {
	var paths []string
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/applications/search", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		_, _ = w.Write([]byte(`{"data":[]}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/applications/search/_cursor", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		_, _ = w.Write([]byte(`{"data":[],"cursor":"next"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT")
	query := url.Values{"limit": []string{"50"}}
	if _, err := c.SearchApplications(context.Background(), "domain-1", false, query); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SearchApplications(context.Background(), "domain-1", true, query); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/applications/search?limit=50",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/applications/search/_cursor?limit=50",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %#v, want %#v", paths, want)
	}
}

func TestApplyDefaults(t *testing.T) {
	t.Parallel()
	model := ApplicationsModel{}
	applyDefaults(&model)
	if model.PaginationMode.ValueString() != "page" || model.Limit.ValueInt64() != 50 || model.Sort.ValueString() != "updatedAt" || model.Direction.ValueString() != "DESC" || model.Page.ValueInt64() != 0 {
		t.Fatalf("defaults = %#v", model)
	}
}
