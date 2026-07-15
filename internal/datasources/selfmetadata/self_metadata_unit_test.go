package selfmetadata

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestSelfMetadataMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewSelfMetadataDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_self_metadata" {
		t.Fatalf("type name = %q, want graviteeam_self_metadata", resp.TypeName)
	}
}

func TestSelfMetadataSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewSelfMetadataDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["kind"]; !attr.IsRequired() {
		t.Fatal("kind should be required")
	}
	if attr := resp.Schema.Attributes["result_json"]; !attr.IsComputed() {
		t.Fatal("result_json should be computed")
	}
	if !strings.Contains(resp.Schema.Attributes["kind"].GetDescription(), "current_user") {
		t.Fatalf("kind description = %q, want supported kind list", resp.Schema.Attributes["kind"].GetDescription())
	}
}

func TestSelfMetadataConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	dataSource := &SelfMetadataDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestSelfMetadataKindListIsSorted(t *testing.T) {
	t.Parallel()

	got := selfMetadataKindList()
	want := "current_user, notifications"
	if got != want {
		t.Fatalf("kind list = %q, want %q", got, want)
	}
}

func TestFormatJSONRawString(t *testing.T) {
	t.Parallel()

	got := formatJSON([]byte("plain"))
	if got != `"plain"` {
		t.Fatalf("formatJSON() = %q, want %q", got, `"plain"`)
	}
}

func TestFormatJSONEmptyBody(t *testing.T) {
	t.Parallel()

	got := formatJSON(nil)
	if got != "null" {
		t.Fatalf("formatJSON() = %q, want null", got)
	}
}

func TestFormatJSONPrettyPrintsObjects(t *testing.T) {
	t.Parallel()

	got := formatJSON([]byte(`{"notifications":[{"id":"n1"}]}`))
	if !strings.Contains(got, "\n") || !strings.Contains(got, `"notifications": [`) || !strings.Contains(got, `"n1"`) {
		t.Fatalf("formatJSON() = %q, want pretty JSON object", got)
	}
}
