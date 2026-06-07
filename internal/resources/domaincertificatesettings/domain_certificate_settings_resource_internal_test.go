package domaincertificatesettings

import (
	"reflect"
	"testing"
)

func TestBuildBodySetsFallbackCertificate(t *testing.T) {
	t.Parallel()

	got := buildBody("cert-1")
	want := map[string]interface{}{"fallbackCertificate": "cert-1"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildDeleteBodyClearsFallbackCertificate(t *testing.T) {
	t.Parallel()

	got := buildDeleteBody()

	if value, ok := got["fallbackCertificate"]; !ok || value != nil {
		t.Fatalf("fallbackCertificate = %#v, want nil", got)
	}
}

func TestReadFallbackCertificateID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		domain map[string]interface{}
		want   string
	}{
		"present": {
			domain: map[string]interface{}{
				"certificateSettings": map[string]interface{}{
					"fallbackCertificate": "cert-1",
				},
			},
			want: "cert-1",
		},
		"missing settings": {
			domain: map[string]interface{}{},
			want:   "",
		},
		"wrong settings type": {
			domain: map[string]interface{}{"certificateSettings": "invalid"},
			want:   "",
		},
		"missing fallback": {
			domain: map[string]interface{}{"certificateSettings": map[string]interface{}{}},
			want:   "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := readFallbackCertificateID(test.domain); got != test.want {
				t.Fatalf("fallback = %q, want %q", got, test.want)
			}
		})
	}
}
