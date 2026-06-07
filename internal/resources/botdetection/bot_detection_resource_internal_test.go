package botdetection

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
	NewBotDetectionResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_bot_detection"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewBotDetectionResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "detection_type", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "configuration", true, false, false)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&BotDetectionResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		id                 string
		wantDomainID       string
		wantBotDetectionID string
		wantOK             bool
	}{
		{
			name:               "valid",
			id:                 "domain-1/bot-1",
			wantDomainID:       "domain-1",
			wantBotDetectionID: "bot-1",
			wantOK:             true,
		},
		{
			name:               "preserves splitN behavior",
			id:                 "domain-1/bot-1/extra",
			wantDomainID:       "domain-1",
			wantBotDetectionID: "bot-1/extra",
			wantOK:             true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotBotDetectionID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotBotDetectionID != tt.wantBotDetectionID {
				t.Fatalf("bot detection ID = %q, want %q", gotBotDetectionID, tt.wantBotDetectionID)
			}
		})
	}
}

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := BotDetectionModel{
		Name:          types.StringValue("reCAPTCHA"),
		Type:          types.StringValue("google-recaptcha-v3-am-bot-detection"),
		DetectionType: types.StringValue("CAPTCHA"),
		Configuration: types.StringValue(`{"siteKey":"site","secretKey":"secret"}`),
	}

	got := buildCreateBody(plan)
	want := map[string]interface{}{
		"name":          "reCAPTCHA",
		"type":          "google-recaptcha-v3-am-bot-detection",
		"detectionType": "CAPTCHA",
		"configuration": `{"siteKey":"site","secretKey":"secret"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("create body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyOmitsReplaceOnlyDetectionType(t *testing.T) {
	t.Parallel()

	plan := BotDetectionModel{
		Name:          types.StringValue("reCAPTCHA"),
		Type:          types.StringValue("google-recaptcha-v3-am-bot-detection"),
		DetectionType: types.StringValue("CAPTCHA"),
		Configuration: types.StringValue(`{"siteKey":"site","secretKey":"secret"}`),
	}

	got := buildUpdateBody(plan)
	want := map[string]interface{}{
		"name":          "reCAPTCHA",
		"type":          "google-recaptcha-v3-am-bot-detection",
		"configuration": `{"siteKey":"site","secretKey":"secret"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("update body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := BotDetectionModel{
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		DetectionType: types.StringValue("old-detection"),
		Configuration: types.StringValue(`{"secretKey":"real-secret"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new-name",
		"type":          "new-type",
		"detectionType": "CAPTCHA",
		"configuration": `{"secretKey":"********"}`,
	})

	if got, want := model.Name.ValueString(), "new-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "new-type"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.DetectionType.ValueString(), "CAPTCHA"; got != want {
		t.Fatalf("detection type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"secretKey":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
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
