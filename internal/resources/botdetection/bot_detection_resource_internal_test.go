package botdetection

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
