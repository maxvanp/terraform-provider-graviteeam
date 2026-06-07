package theme

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
	NewThemeResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_theme"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewThemeResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["domain_id"]; attr == nil || !attr.IsRequired() {
		t.Fatalf("domain_id should be required")
	}
	for _, name := range []string{
		"logo_url",
		"logo_width",
		"favicon_url",
		"primary_button_color_hex",
		"secondary_button_color_hex",
		"primary_text_color_hex",
		"secondary_text_color_hex",
		"css",
	} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; attr == nil || !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ThemeResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	plan := ThemeModel{
		LogoURL:                 types.StringValue("https://example.com/logo.png"),
		LogoWidth:               types.Int64Value(180),
		FaviconURL:              types.StringValue("https://example.com/favicon.ico"),
		PrimaryButtonColorHex:   types.StringValue("#111111"),
		SecondaryButtonColorHex: types.StringValue("#222222"),
		PrimaryTextColorHex:     types.StringValue("#333333"),
		SecondaryTextColorHex:   types.StringValue("#444444"),
		CSS:                     types.StringValue("body { color: #333; }"),
	}

	got := resource.buildBody(plan)
	want := map[string]interface{}{
		"logoUrl":                 "https://example.com/logo.png",
		"logoWidth":               int64(180),
		"faviconUrl":              "https://example.com/favicon.ico",
		"primaryButtonColorHex":   "#111111",
		"secondaryButtonColorHex": "#222222",
		"primaryTextColorHex":     "#333333",
		"secondaryTextColorHex":   "#444444",
		"css":                     "body { color: #333; }",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyMergesCurrentAndClearsRemovedFields(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	current := map[string]interface{}{
		"id":                      "theme-1",
		"logoUrl":                 "https://example.com/old-logo.png",
		"logoWidth":               float64(120),
		"faviconUrl":              "https://example.com/old.ico",
		"primaryButtonColorHex":   "#000000",
		"secondaryButtonColorHex": "#010101",
		"primaryTextColorHex":     "#020202",
		"secondaryTextColorHex":   "#030303",
		"css":                     ".old {}",
		"apiManaged":              "preserve-me",
	}
	state := ThemeModel{
		LogoURL:                 types.StringValue("https://example.com/old-logo.png"),
		LogoWidth:               types.Int64Value(120),
		FaviconURL:              types.StringValue("https://example.com/old.ico"),
		PrimaryButtonColorHex:   types.StringValue("#000000"),
		SecondaryButtonColorHex: types.StringValue("#010101"),
		PrimaryTextColorHex:     types.StringValue("#020202"),
		SecondaryTextColorHex:   types.StringValue("#030303"),
		CSS:                     types.StringValue(".old {}"),
	}
	plan := ThemeModel{
		LogoURL:                 types.StringValue("https://example.com/new-logo.png"),
		LogoWidth:               types.Int64Null(),
		FaviconURL:              types.StringNull(),
		PrimaryButtonColorHex:   types.StringValue("#111111"),
		SecondaryButtonColorHex: types.StringNull(),
		PrimaryTextColorHex:     types.StringValue("#222222"),
		SecondaryTextColorHex:   types.StringNull(),
		CSS:                     types.StringValue(".new {}"),
	}

	got := resource.buildUpdateBody(plan, state, current)
	want := map[string]interface{}{
		"id":                      "theme-1",
		"logoUrl":                 "https://example.com/new-logo.png",
		"logoWidth":               0,
		"faviconUrl":              "",
		"primaryButtonColorHex":   "#111111",
		"secondaryButtonColorHex": "",
		"primaryTextColorHex":     "#222222",
		"secondaryTextColorHex":   "",
		"css":                     ".new {}",
		"apiManaged":              "preserve-me",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	model := ThemeModel{}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":                      "theme-1",
		"logoUrl":                 "https://example.com/logo.png",
		"logoWidth":               float64(180),
		"faviconUrl":              "https://example.com/favicon.ico",
		"primaryButtonColorHex":   "#111111",
		"secondaryButtonColorHex": "#222222",
		"primaryTextColorHex":     "#333333",
		"secondaryTextColorHex":   "#444444",
		"css":                     ".theme {}",
	})

	if got, want := model.ID.ValueString(), "theme-1"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.LogoURL.ValueString(), "https://example.com/logo.png"; got != want {
		t.Fatalf("logo URL = %q, want %q", got, want)
	}
	if got, want := model.LogoWidth.ValueInt64(), int64(180); got != want {
		t.Fatalf("logo width = %d, want %d", got, want)
	}
	if got, want := model.FaviconURL.ValueString(), "https://example.com/favicon.ico"; got != want {
		t.Fatalf("favicon URL = %q, want %q", got, want)
	}
	if got, want := model.PrimaryButtonColorHex.ValueString(), "#111111"; got != want {
		t.Fatalf("primary button color = %q, want %q", got, want)
	}
	if got, want := model.SecondaryButtonColorHex.ValueString(), "#222222"; got != want {
		t.Fatalf("secondary button color = %q, want %q", got, want)
	}
	if got, want := model.PrimaryTextColorHex.ValueString(), "#333333"; got != want {
		t.Fatalf("primary text color = %q, want %q", got, want)
	}
	if got, want := model.SecondaryTextColorHex.ValueString(), "#444444"; got != want {
		t.Fatalf("secondary text color = %q, want %q", got, want)
	}
	if got, want := model.CSS.ValueString(), ".theme {}"; got != want {
		t.Fatalf("css = %q, want %q", got, want)
	}
}

func TestReadIntoModelClearsEmptyValues(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	model := ThemeModel{
		LogoURL:               types.StringValue("old"),
		LogoWidth:             types.Int64Value(10),
		FaviconURL:            types.StringValue("old"),
		PrimaryButtonColorHex: types.StringValue("old"),
		CSS:                   types.StringValue("old"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"logoUrl":               "",
		"logoWidth":             float64(0),
		"faviconUrl":            "",
		"primaryButtonColorHex": "",
		"css":                   "",
	})

	if !model.LogoURL.IsNull() {
		t.Fatalf("logo URL should be null, got %q", model.LogoURL.ValueString())
	}
	if !model.LogoWidth.IsNull() {
		t.Fatalf("logo width should be null, got %d", model.LogoWidth.ValueInt64())
	}
	if !model.FaviconURL.IsNull() {
		t.Fatalf("favicon URL should be null, got %q", model.FaviconURL.ValueString())
	}
	if !model.PrimaryButtonColorHex.IsNull() {
		t.Fatalf("primary button color should be null, got %q", model.PrimaryButtonColorHex.ValueString())
	}
	if !model.CSS.IsNull() {
		t.Fatalf("css should be null, got %q", model.CSS.ValueString())
	}
}
