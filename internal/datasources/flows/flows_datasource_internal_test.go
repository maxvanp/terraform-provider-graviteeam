package flows

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestFlowsMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewFlowsDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_flows" {
		t.Fatalf("type name = %q, want graviteeam_flows", resp.TypeName)
	}
}

func TestFlowsSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewFlowsDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["domain_id"]; !attr.IsRequired() {
		t.Fatal("domain_id should be required")
	}
	if attr := resp.Schema.Attributes["flows"]; !attr.IsComputed() {
		t.Fatal("flows should be computed")
	}
}

func TestFlowsConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	ds := &FlowsDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestFormatFlowsProducesStableIndentedJSON(t *testing.T) {
	t.Parallel()

	got, err := formatFlows([]interface{}{
		map[string]interface{}{"id": "login", "enabled": true},
	})
	if err != nil {
		t.Fatalf("format flows: %v", err)
	}
	want := "[\n  {\n    \"enabled\": true,\n    \"id\": \"login\"\n  }\n]"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestFormatFlowsReturnsMarshalError(t *testing.T) {
	t.Parallel()

	_, err := formatFlows([]interface{}{func() {}})
	if err == nil {
		t.Fatalf("expected marshal error")
	}
}
