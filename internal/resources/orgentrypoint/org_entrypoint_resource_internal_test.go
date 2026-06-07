package orgentrypoint

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewOrgEntrypointResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_org_entrypoint"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewOrgEntrypointResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"name", "url", "tags"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["description"]; attr == nil || !attr.IsOptional() {
		t.Fatalf("description should be optional")
	}
	for _, name := range []string{"id", "default_entrypoint"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsComputed() {
			t.Fatalf("attribute %q should be computed", name)
		}
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&OrgEntrypointResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildBodyForCreate(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{
		Name:        types.StringValue("entrypoint-1"),
		Description: types.StringValue("description"),
		URL:         types.StringValue("https://login.example.com"),
		Tags: []types.String{
			types.StringValue("tag-1"),
			types.StringValue("tag-2"),
		},
	}

	got := buildBody(model)
	want := map[string]interface{}{
		"name":        "entrypoint-1",
		"description": "description",
		"url":         "https://login.example.com",
		"tags":        []string{"tag-1", "tag-2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyClearsRemovedDescriptionForUpdate(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{
		Name:        types.StringValue("entrypoint-1"),
		Description: types.StringNull(),
		URL:         types.StringValue("https://login.example.com"),
		Tags:        []types.String{types.StringValue("tag-1")},
	}
	state := OrgEntrypointModel{
		Description: types.StringValue("old description"),
	}

	got := buildBody(model, state)
	want := map[string]interface{}{
		"name":        "entrypoint-1",
		"description": "",
		"url":         "https://login.example.com",
		"tags":        []string{"tag-1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodyOmitsAbsentDescriptionForCreate(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{
		Name:        types.StringValue("entrypoint-1"),
		Description: types.StringNull(),
		URL:         types.StringValue("https://login.example.com"),
		Tags:        []types.String{},
	}

	got := buildBody(model)
	want := map[string]interface{}{
		"name": "entrypoint-1",
		"url":  "https://login.example.com",
		"tags": []string{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelMapsEntrypoint(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{}

	readIntoModel(&model, map[string]interface{}{
		"id":                "entrypoint-id",
		"name":              "entrypoint-1",
		"description":       "description",
		"url":               "https://login.example.com",
		"defaultEntrypoint": true,
		"tags":              []interface{}{"tag-1", "tag-2"},
	})

	if got, want := model.ID.ValueString(), "entrypoint-id"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.Name.ValueString(), "entrypoint-1"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Description.ValueString(), "description"; got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
	if got, want := model.URL.ValueString(), "https://login.example.com"; got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
	if !model.DefaultEntrypoint.ValueBool() {
		t.Fatalf("default entrypoint should be true")
	}
	if got, want := stringValues(model.Tags), []string{"tag-1", "tag-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelClearsEmptyDescriptionAndUnknownDefault(t *testing.T) {
	t.Parallel()

	model := OrgEntrypointModel{
		Description:       types.StringValue("old description"),
		DefaultEntrypoint: types.BoolValue(true),
	}

	readIntoModel(&model, map[string]interface{}{
		"description": "",
	})

	if !model.Description.IsNull() {
		t.Fatalf("description should be null, got %q", model.Description.ValueString())
	}
	if !model.DefaultEntrypoint.IsNull() {
		t.Fatalf("default entrypoint should be null")
	}
}

func TestInterfaceStringsSkipsNonStringEntries(t *testing.T) {
	t.Parallel()

	got := stringValues(interfaceStrings([]interface{}{"tag-1", 42, "tag-2"}))
	want := []string{"tag-1", "tag-2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
}
