package formpreview

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewFormPreviewDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_form_preview"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewFormPreviewDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "template", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "content", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "result_json", false, false, true)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&FormPreviewDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestFormatJSONHandlesEmptyJSONAndText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty": {nil, "null"},
		"text":  {[]byte("<html>preview</html>"), `"\u003chtml\u003epreview\u003c/html\u003e"`},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := formatJSON(test.raw)
			if err != nil {
				t.Fatalf("format json: %v", err)
			}
			if got != test.want {
				t.Fatalf("json = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatJSONProducesStableIndentedPreviewJSON(t *testing.T) {
	t.Parallel()

	got, err := formatJSON([]byte(`{"content":"ok","type":"FORM"}`))
	if err != nil {
		t.Fatalf("format json: %v", err)
	}
	want := "{\n  \"content\": \"ok\",\n  \"type\": \"FORM\"\n}"
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
