package usercollections

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestCollectionDataSourceMetadataAndSchema(t *testing.T) {
	t.Parallel()

	ds := &collectionDataSource{
		typeSuffix:  "user_credentials",
		resultName:  "credentials",
		description: "Reads credentials for a Gravitee AM user",
	}

	var metadataResp datasource.MetadataResponse
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "graviteeam"}, &metadataResp)
	if metadataResp.TypeName != "graviteeam_user_credentials" {
		t.Fatalf("type name = %q, want graviteeam_user_credentials", metadataResp.TypeName)
	}

	var schemaResp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	for _, name := range []string{"domain_id", "user_id"} {
		attr, ok := schemaResp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := schemaResp.Schema.Attributes["items_json"]; !attr.IsComputed() {
		t.Fatal("items_json should be computed")
	}
}

func TestCollectionDataSourceConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	ds := &collectionDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestCollectionDataSourceConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	ds := &collectionDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if ds.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestCollectionDataSourceConfigureIgnoresNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &collectionDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if ds.client != nil {
		t.Fatal("expected nil provider data to leave client unset")
	}
}

func TestUserCollectionWrapperMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ds   datasource.DataSource
		want string
	}{
		{name: "consents", ds: NewUserConsentsDataSource(), want: "graviteeam_user_consents"},
		{name: "credentials", ds: NewUserCredentialsDataSource(), want: "graviteeam_user_credentials"},
		{name: "devices", ds: NewUserDevicesDataSource(), want: "graviteeam_user_devices"},
		{name: "factors", ds: NewUserFactorsDataSource(), want: "graviteeam_user_factors"},
		{name: "identities", ds: NewUserIdentitiesDataSource(), want: "graviteeam_user_identities"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var resp datasource.MetadataResponse
			tt.ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "graviteeam"}, &resp)
			if resp.TypeName != tt.want {
				t.Fatalf("type name = %q, want %q", resp.TypeName, tt.want)
			}
		})
	}
}

func TestFormatCollectionItemsProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatCollectionItems([]byte(`[{"id":"item-1","type":"credential"}]`))
	if err != nil {
		t.Fatalf("format collection items: %v", err)
	}
	want := "[\n  {\n    \"id\": \"item-1\",\n    \"type\": \"credential\"\n  }\n]"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFormatCollectionItemsRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := formatCollectionItems([]byte(`{`))
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestCollectionDataSourceReadFormatsRemoteItems(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/credentials", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write([]byte(`[{"id":"credential-1","type":"password"}]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &collectionDataSource{
		client:     client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
		typeSuffix: "user_credentials",
		collection: "credentials",
		resultName: "credentials",
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := collectionConfig(schemaResp.Schema, collectionModel{
		DomainID: types.StringValue("domain-123"),
		UserID:   types.StringValue("user-123"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var state collectionModel
	if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get state: %#v", diags)
	}
	want := "[\n  {\n    \"id\": \"credential-1\",\n    \"type\": \"password\"\n  }\n]"
	if state.Items.ValueString() != want {
		t.Fatalf("items_json = %q, want %q", state.Items.ValueString(), want)
	}
}

func TestCollectionDataSourceReadReportsRemoteAndFormatErrors(t *testing.T) {
	tests := map[string]struct {
		status int
		body   string
	}{
		"remote error": {status: http.StatusInternalServerError, body: "collection failed"},
		"invalid json": {status: http.StatusOK, body: "{"},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/users/user-123/devices", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			dataSource := &collectionDataSource{
				client:     client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
				typeSuffix: "user_devices",
				collection: "devices",
				resultName: "devices",
			}
			var schemaResp datasource.SchemaResponse
			dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
			config := collectionConfig(schemaResp.Schema, collectionModel{
				DomainID: types.StringValue("domain-123"),
				UserID:   types.StringValue("user-123"),
			})

			readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected diagnostics")
			}
		})
	}
}

func collectionConfig(schema datasourceschema.Schema, model collectionModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id":  tftypes.String,
				"user_id":    tftypes.String,
				"items_json": tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id":  tftypes.NewValue(tftypes.String, model.DomainID.ValueString()),
				"user_id":    tftypes.NewValue(tftypes.String, model.UserID.ValueString()),
				"items_json": tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}
