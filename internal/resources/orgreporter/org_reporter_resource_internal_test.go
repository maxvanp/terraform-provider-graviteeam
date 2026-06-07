package orgreporter

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
