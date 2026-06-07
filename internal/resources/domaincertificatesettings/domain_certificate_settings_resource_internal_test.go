package domaincertificatesettings

import (
	"reflect"
	"testing"
)

func TestBuildBody(t *testing.T) {
	t.Parallel()

	got := buildBody("cert-1")
	want := map[string]interface{}{
		"fallbackCertificate": "cert-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildDeleteBody(t *testing.T) {
	t.Parallel()

	got := buildDeleteBody()
	want := map[string]interface{}{
		"fallbackCertificate": nil,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadFallbackCertificateID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		domain map[string]interface{}
		want   string
	}{
		{
			name: "present",
			domain: map[string]interface{}{
				"certificateSettings": map[string]interface{}{
					"fallbackCertificate": "cert-1",
				},
			},
			want: "cert-1",
		},
		{
			name: "missing certificate settings",
			domain: map[string]interface{}{
				"name": "domain-1",
			},
			want: "",
		},
		{
			name: "missing fallback certificate",
			domain: map[string]interface{}{
				"certificateSettings": map[string]interface{}{},
			},
			want: "",
		},
		{
			name: "wrong fallback type",
			domain: map[string]interface{}{
				"certificateSettings": map[string]interface{}{
					"fallbackCertificate": 123,
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := readFallbackCertificateID(tt.domain); got != tt.want {
				t.Fatalf("fallback certificate ID = %q, want %q", got, tt.want)
			}
		})
	}
}
