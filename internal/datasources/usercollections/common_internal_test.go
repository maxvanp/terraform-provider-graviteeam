package usercollections

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
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
