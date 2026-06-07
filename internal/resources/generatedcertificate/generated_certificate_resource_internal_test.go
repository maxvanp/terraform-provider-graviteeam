package generatedcertificate

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestReadGeneratedCertificateMapsReturnedFields(t *testing.T) {
	t.Parallel()

	model := GeneratedCertificateModel{}

	readGeneratedCertificate(&model, map[string]interface{}{
		"name": "system-generated-cert",
		"type": "pem",
	})

	if got, want := model.Name.ValueString(), "system-generated-cert"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "pem"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
}

func TestReadGeneratedCertificatePreservesMissingFields(t *testing.T) {
	t.Parallel()

	model := GeneratedCertificateModel{
		Name: types.StringValue("existing-name"),
		Type: types.StringValue("existing-type"),
	}

	readGeneratedCertificate(&model, map[string]interface{}{})

	if got, want := model.Name.ValueString(), "existing-name"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Type.ValueString(), "existing-type"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
}

func TestCertificateIDReturnsID(t *testing.T) {
	t.Parallel()

	got, ok := certificateID(map[string]interface{}{"id": "cert-id"})
	if !ok {
		t.Fatalf("expected id")
	}
	if got != "cert-id" {
		t.Fatalf("id = %q, want cert-id", got)
	}
}

func TestCertificateIDRejectsMissingEmptyOrMalformedID(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]interface{}{
		"missing":   {},
		"empty":     {"id": ""},
		"malformed": {"id": 42},
	}

	for name, result := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if id, ok := certificateID(result); ok {
				t.Fatalf("id = %q, want not ok", id)
			}
		})
	}
}
