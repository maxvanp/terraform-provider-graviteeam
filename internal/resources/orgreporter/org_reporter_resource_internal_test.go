package orgreporter

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewOrgReporterResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_reporter"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgReporterResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "configuration", true, false, false)
	assertBoolAttribute(t, resp.Schema.Attributes, "enabled", false, true, true)
	assertBoolAttribute(t, resp.Schema.Attributes, "inherited", false, true, true)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgReporterResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestBuildBodyUsesPlannedOrganizationReporterFields(t *testing.T) {
	t.Parallel()

	plan := OrgReporterModel{
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{"directory":"/tmp"}`),
		Enabled:       types.BoolValue(true),
		Inherited:     types.BoolValue(false),
	}

	got := buildBody(plan, map[string]interface{}{"inherited": true})
	want := map[string]interface{}{
		"name":          "file reporter",
		"type":          "reporter-am-file",
		"configuration": `{"directory":"/tmp"}`,
		"enabled":       true,
		"inherited":     false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyPreservesCurrentInheritedWhenPlanUnknown(t *testing.T) {
	t.Parallel()

	plan := OrgReporterModel{
		Name:          types.StringValue("file reporter"),
		Type:          types.StringValue("reporter-am-file"),
		Configuration: types.StringValue(`{}`),
		Enabled:       types.BoolValue(false),
		Inherited:     types.BoolUnknown(),
	}

	got := buildBody(plan, map[string]interface{}{"inherited": true})

	if got["inherited"] != true {
		t.Fatalf("inherited = %#v, want preserved true", got["inherited"])
	}
}

func TestReadIntoModelMapsOrganizationReporterFieldsAndPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := OrgReporterModel{
		Configuration: types.StringValue(`{"secret":"planned"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "updated",
		"type":          "reporter-am-file",
		"configuration": `{"secret":"***"}`,
		"enabled":       false,
		"inherited":     true,
	})

	if model.Name.ValueString() != "updated" ||
		model.Type.ValueString() != "reporter-am-file" ||
		model.Enabled.ValueBool() ||
		!model.Inherited.ValueBool() {
		t.Fatalf("model fields not mapped: %#v", model)
	}
	if model.Configuration.ValueString() != `{"secret":"planned"}` {
		t.Fatalf("configuration = %q, want preserved", model.Configuration.ValueString())
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

func assertBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.BoolAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}
