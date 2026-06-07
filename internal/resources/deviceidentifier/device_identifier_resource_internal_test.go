package deviceidentifier

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		id                     string
		wantDomainID           string
		wantDeviceIdentifierID string
		wantOK                 bool
	}{
		{
			name:                   "valid",
			id:                     "domain-1/device-1",
			wantDomainID:           "domain-1",
			wantDeviceIdentifierID: "device-1",
			wantOK:                 true,
		},
		{
			name:                   "preserves splitN behavior",
			id:                     "domain-1/device-1/extra",
			wantDomainID:           "domain-1",
			wantDeviceIdentifierID: "device-1/extra",
			wantOK:                 true,
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

			gotDomainID, gotDeviceIdentifierID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotDeviceIdentifierID != tt.wantDeviceIdentifierID {
				t.Fatalf("device identifier ID = %q, want %q", gotDeviceIdentifierID, tt.wantDeviceIdentifierID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := DeviceIdentifierModel{
		Name:          types.StringValue("Fingerprint"),
		Type:          types.StringValue("fingerprintjs-v3-am-device-identifier"),
		Configuration: types.StringValue(`{"browserToken":"secret"}`),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":          "Fingerprint",
		"type":          "fingerprintjs-v3-am-device-identifier",
		"configuration": `{"browserToken":"secret"}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModelPreservesConfiguration(t *testing.T) {
	t.Parallel()

	model := DeviceIdentifierModel{
		Name:          types.StringValue("old-name"),
		Type:          types.StringValue("old-type"),
		Configuration: types.StringValue(`{"browserToken":"real-secret"}`),
	}

	readIntoModel(&model, map[string]interface{}{
		"name":          "new-name",
		"type":          "fingerprintjs-v3-am-device-identifier",
		"configuration": `{"browserToken":"********"}`,
	})

	if got, want := model.Name.ValueString(), "new-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "fingerprintjs-v3-am-device-identifier"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if got, want := model.Configuration.ValueString(), `{"browserToken":"real-secret"}`; got != want {
		t.Fatalf("configuration = %q, want preserved %q", got, want)
	}
}
