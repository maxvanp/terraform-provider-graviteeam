package orgtag

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
