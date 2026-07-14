package trustdomain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadataSchemaAndConfigure(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &TrustDomainResource{}
	var metadataResp resource.MetadataResponse
	resourceUnderTest.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "graviteeam"}, &metadataResp)
	if metadataResp.TypeName != "graviteeam_trust_domain" {
		t.Fatalf("type name = %q", metadataResp.TypeName)
	}

	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	for _, name := range []string{"domain_id", "name", "bundle_source", "jwks_url"} {
		if attr := schemaResp.Schema.Attributes[name]; attr == nil || !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"id", "created_at", "updated_at"} {
		if attr := schemaResp.Schema.Attributes[name]; attr == nil || !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &resource.ConfigureResponse{})
	if resourceUnderTest.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestConfigureValidationAndImportParsing(t *testing.T) {
	t.Parallel()

	var configureResp resource.ConfigureResponse
	(&TrustDomainResource{}).Configure(context.Background(), resource.ConfigureRequest{ProviderData: "wrong"}, &configureResp)
	if !configureResp.Diagnostics.HasError() {
		t.Fatal("expected invalid configure diagnostic")
	}

	domainID, trustDomainID, ok := parseImportID("domain-1/trust-1")
	if !ok || domainID != "domain-1" || trustDomainID != "trust-1" {
		t.Fatalf("parseImportID returned %q, %q, %t", domainID, trustDomainID, ok)
	}
	for _, invalid := range []string{"domain-only", "/trust", "domain/"} {
		if _, _, ok := parseImportID(invalid); ok {
			t.Fatalf("expected %q to be invalid", invalid)
		}
	}
}

func TestBuildBodyAndReadIntoModel(t *testing.T) {
	t.Parallel()

	model := TrustDomainModel{
		Name:                   types.StringValue("spiffe.example.com"),
		Description:            types.StringValue("Workloads"),
		BundleSource:           types.StringValue("JWKS_URL"),
		JWKsURL:                types.StringValue("https://example.com/jwks.json"),
		RefreshIntervalSeconds: types.Int64Value(300),
		AllowedAlgorithms:      []types.String{types.StringValue("RS256"), types.StringValue("ES256")},
	}
	want := map[string]interface{}{
		"name": "spiffe.example.com", "description": "Workloads", "bundleSource": "JWKS_URL",
		"jwksUrl": "https://example.com/jwks.json", "refreshIntervalSeconds": int64(300),
		"allowedAlgorithms": []string{"RS256", "ES256"},
	}
	if got := buildBody(model, true); !reflect.DeepEqual(got, want) {
		t.Fatalf("create body = %#v, want %#v", got, want)
	}
	if got := buildBody(model, false); got["name"] != nil {
		t.Fatalf("update body should omit name: %#v", got)
	}

	(&TrustDomainResource{}).readIntoModel(&model, map[string]interface{}{
		"id": "trust-1", "name": "spiffe.example.com", "bundleSource": "jwks_url",
		"jwksUrl": "https://example.com/jwks.json", "refreshIntervalSeconds": float64(600),
		"allowedAlgorithms": []interface{}{"RS256"}, "createdAt": float64(100), "updatedAt": int64(200),
	})
	if model.ID.ValueString() != "trust-1" || model.BundleSource.ValueString() != "JWKS_URL" || model.RefreshIntervalSeconds.ValueInt64() != 600 {
		t.Fatalf("unexpected model: %#v", model)
	}
	if len(model.AllowedAlgorithms) != 1 || model.AllowedAlgorithms[0].ValueString() != "RS256" {
		t.Fatalf("allowed algorithms = %#v", model.AllowedAlgorithms)
	}
}

func TestTrustDomainClientCRUD(t *testing.T) {
	var methods []string
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/trust-domains", func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"trust-1","name":"spiffe.example.com"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-1/trust-domains/trust-1", func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"id":"trust-1","name":"spiffe.example.com"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT")
	ctx := context.Background()
	if _, err := c.CreateTrustDomain(ctx, "domain-1", map[string]interface{}{"name": "spiffe.example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetTrustDomain(ctx, "domain-1", "trust-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateTrustDomain(ctx, "domain-1", "trust-1", map[string]interface{}{"description": "updated"}); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteTrustDomain(ctx, "domain-1", "trust-1"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"POST", "GET", "PUT", "DELETE"}; !reflect.DeepEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
}
