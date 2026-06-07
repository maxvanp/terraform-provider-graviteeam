package authdevicenotifier

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		id             string
		wantDomainID   string
		wantNotifierID string
		wantOK         bool
	}{
		{
			name:           "valid",
			id:             "domain-1/notifier-1",
			wantDomainID:   "domain-1",
			wantNotifierID: "notifier-1",
			wantOK:         true,
		},
		{
			name:           "preserves splitN behavior",
			id:             "domain-1/notifier-1/extra",
			wantDomainID:   "domain-1",
			wantNotifierID: "notifier-1/extra",
			wantOK:         true,
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

			gotDomainID, gotNotifierID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotNotifierID != tt.wantNotifierID {
				t.Fatalf("notifier ID = %q, want %q", gotNotifierID, tt.wantNotifierID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := AuthDeviceNotifierModel{
		Name:          types.StringValue("HTTP Notifier"),
		Type:          types.StringValue("http-am-authdevice-notifier"),
		Configuration: types.StringValue(`{"endpoint":"https://example.com","headerValue":"secret"}`),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":          "HTTP Notifier",
		"type":          "http-am-authdevice-notifier",
		"configuration": `{"endpoint":"https://example.com","headerValue":"secret"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := AuthDeviceNotifierModel{
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		Configuration: types.StringValue(`{"headerValue":"real-secret"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new-name",
		"type":          "http-am-authdevice-notifier",
		"configuration": `{"headerValue":"********"}`,
	})

	if got, want := model.Name.ValueString(), "new-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "http-am-authdevice-notifier"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"headerValue":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}
}
