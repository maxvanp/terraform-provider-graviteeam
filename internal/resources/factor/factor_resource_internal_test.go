package factor

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFactorMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewFactorResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_factor" {
		t.Fatalf("type name = %q, want graviteeam_factor", resp.TypeName)
	}
}

func TestFactorSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewFactorResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "name", "factor_type"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; !attr.IsComputed() {
		t.Fatal("id should be computed")
	}
}

func TestFactorConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &FactorResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestBuildCreateBody(t *testing.T) {
	t.Parallel()

	plan := FactorModel{
		Name:       types.StringValue("Login TOTP"),
		FactorType: types.StringValue("TOTP"),
	}

	got := buildCreateBody(plan, "otp-am-factor")
	want := map[string]interface{}{
		"name":          "Login TOTP",
		"type":          "otp-am-factor",
		"factorType":    "TOTP",
		"configuration": "{}",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyOmitsImmutableFactorType(t *testing.T) {
	t.Parallel()

	plan := FactorModel{
		Name:       types.StringValue("Updated TOTP"),
		FactorType: types.StringValue("TOTP"),
	}

	got := buildUpdateBody(plan, "otp-am-factor")
	want := map[string]interface{}{
		"name":          "Updated TOTP",
		"type":          "otp-am-factor",
		"configuration": "{}",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
	if _, ok := got["factorType"]; ok {
		t.Fatalf("factorType should be omitted from update body")
	}
}

func TestReadIntoModelMapsLowercaseAPIFactorType(t *testing.T) {
	t.Parallel()

	model := FactorModel{}

	readIntoModel(&model, map[string]interface{}{
		"name":       "Login Email",
		"factorType": "email",
	})

	if got, want := model.Name.ValueString(), "Login Email"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.FactorType.ValueString(), "EMAIL"; got != want {
		t.Fatalf("factorType = %q, want %q", got, want)
	}
}

func TestReadIntoModelFallsBackToPluginType(t *testing.T) {
	t.Parallel()

	model := FactorModel{}

	readIntoModel(&model, map[string]interface{}{
		"type": "sms-am-factor",
	})

	if got, want := model.FactorType.ValueString(), "SMS"; got != want {
		t.Fatalf("factorType = %q, want %q", got, want)
	}
}

func TestReadIntoModelPreservesUnknownAPIFactorType(t *testing.T) {
	t.Parallel()

	model := FactorModel{}

	readIntoModel(&model, map[string]interface{}{
		"factorType": "custom",
	})

	if got, want := model.FactorType.ValueString(), "custom"; got != want {
		t.Fatalf("factorType = %q, want %q", got, want)
	}
}

func TestFactorTypeValidator(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"TOTP", "EMAIL", "SMS"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			resp := validator.StringResponse{}
			factorTypeValidator{}.ValidateString(context.Background(), validator.StringRequest{
				ConfigValue: types.StringValue(value),
			}, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("expected %s to be accepted, got %v", value, resp.Diagnostics)
			}
		})
	}
}

func TestFactorTypeValidatorRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	resp := validator.StringResponse{}
	factorTypeValidator{}.ValidateString(context.Background(), validator.StringRequest{
		ConfigValue: types.StringValue("PUSH"),
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected unknown factor type to be rejected")
	}
}
