package alerttrigger

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewAlertTriggerResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_alert_trigger"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewAlertTriggerResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "type"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	for _, name := range []string{"enabled", "alert_notifier_ids"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Fatalf("attribute %q should be optional+computed", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; attr == nil || !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&AlertTriggerResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestValidateTriggerTypeNormalizesAcceptedValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"too_many_login_failures", "TOO_MANY_LOGIN_FAILURES", "risk_assessment", "RISK_ASSESSMENT"} {
		if err := validateTriggerType(value); err != nil {
			t.Fatalf("validateTriggerType(%q) returned error: %v", value, err)
		}
	}
}

func TestValidateTriggerTypeRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	if err := validateTriggerType("unknown"); err == nil {
		t.Fatal("validateTriggerType(unknown) returned nil, want error")
	}
}

func TestFindTriggerMatchesNormalizedType(t *testing.T) {
	triggers := []map[string]interface{}{
		{"type": "RISK_ASSESSMENT", "enabled": false},
		{"type": "too_many_login_failures", "enabled": true},
	}

	trigger, ok := findTrigger(triggers, "TOO_MANY_LOGIN_FAILURES")
	if !ok {
		t.Fatal("findTrigger did not find normalized trigger")
	}
	if enabled, _ := trigger["enabled"].(bool); !enabled {
		t.Fatalf("enabled = false, want true")
	}
}

func TestFindTriggerReturnsFalseForMissingType(t *testing.T) {
	if _, ok := findTrigger([]map[string]interface{}{{"type": "RISK_ASSESSMENT"}}, "TOO_MANY_LOGIN_FAILURES"); ok {
		t.Fatal("findTrigger returned true for missing trigger")
	}
}

func TestStringValuesSkipsNullUnknownAndPreservesValues(t *testing.T) {
	setValue, diags := types.SetValue(
		types.StringType,
		[]attr.Value{
			types.StringValue("notifier-1"),
			types.StringNull(),
			types.StringUnknown(),
			types.StringValue("notifier-2"),
		},
	)
	if diags.HasError() {
		t.Fatalf("build set value: %v", diags)
	}

	values := stringValues(setValue)

	if len(values) != 2 || values[0] != "notifier-1" || values[1] != "notifier-2" {
		t.Fatalf("values = %#v, want notifier-1,notifier-2", values)
	}
}

func TestStringValuesReturnsEmptyForNullOrUnknownSet(t *testing.T) {
	for _, setValue := range []types.Set{
		types.SetNull(types.StringType),
		types.SetUnknown(types.StringType),
	} {
		if values := stringValues(setValue); len(values) != 0 {
			t.Fatalf("values = %#v, want empty", values)
		}
	}
}

func TestReadIntoModelMapsAlertTriggerResponse(t *testing.T) {
	model := AlertTriggerModel{
		DomainID: types.StringValue("domain-id"),
		Type:     types.StringValue("risk_assessment"),
	}

	readIntoModel(&model, map[string]interface{}{
		"type":           "too_many_login_failures",
		"enabled":        true,
		"alertNotifiers": []interface{}{"notifier-1", "notifier-2"},
	})

	if model.ID.ValueString() != "domain-id/TOO_MANY_LOGIN_FAILURES" {
		t.Fatalf("id = %q, want domain-id/TOO_MANY_LOGIN_FAILURES", model.ID.ValueString())
	}
	if model.Type.ValueString() != "TOO_MANY_LOGIN_FAILURES" {
		t.Fatalf("type = %q, want TOO_MANY_LOGIN_FAILURES", model.Type.ValueString())
	}
	if !model.Enabled.ValueBool() {
		t.Fatal("enabled = false, want true")
	}
	values := stringValues(model.AlertNotifierIDs)
	if len(values) != 2 || values[0] != "notifier-1" || values[1] != "notifier-2" {
		t.Fatalf("alert notifier ids = %#v, want notifier-1,notifier-2", values)
	}
}

func TestReadIntoModelSetsEmptyNotifierSetWhenAPIOmitsValues(t *testing.T) {
	model := AlertTriggerModel{
		DomainID: types.StringValue("domain-id"),
		Type:     types.StringValue("risk_assessment"),
	}

	readIntoModel(&model, map[string]interface{}{
		"enabled": false,
	})

	if model.ID.ValueString() != "domain-id/RISK_ASSESSMENT" {
		t.Fatalf("id = %q, want domain-id/RISK_ASSESSMENT", model.ID.ValueString())
	}
	if model.Type.ValueString() != "RISK_ASSESSMENT" {
		t.Fatalf("type = %q, want RISK_ASSESSMENT", model.Type.ValueString())
	}
	if !model.AlertNotifierIDs.IsNull() && len(model.AlertNotifierIDs.Elements()) != 0 {
		t.Fatalf("alert notifier ids = %#v, want empty set", model.AlertNotifierIDs)
	}
}
