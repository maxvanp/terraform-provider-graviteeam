package domainflow

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDomainFlowMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewDomainFlowResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if resp.TypeName != "graviteeam_domain_flow" {
		t.Fatalf("type name = %q, want graviteeam_domain_flow", resp.TypeName)
	}
}

func TestDomainFlowSchemaDeclaresRequiredAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewDomainFlowResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"domain_id", "flows"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsRequired() {
			t.Fatalf("attribute %q should be required", name)
		}
	}
	if _, ok := resp.Schema.Attributes["application_id"]; ok {
		t.Fatal("domain flow schema should not expose application_id")
	}
	if !strings.Contains(resp.Schema.Description, "GET/PUT") {
		t.Fatalf("schema description = %q, want GET/PUT ownership hint", resp.Schema.Description)
	}
}

func TestDomainFlowConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &DomainFlowResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestDomainFlowConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&DomainFlowResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestDecodeFlowsParsesCompleteFlowList(t *testing.T) {
	t.Parallel()

	diag := &fakeDiagnostics{}

	got, ok := decodeFlows(`[{"id":"login","name":"Login","enabled":true},{"id":"mfa","enabled":false}]`, diag)
	if !ok {
		t.Fatalf("expected decode success, got diagnostics %#v", diag.errors)
	}
	if len(got) != 2 {
		t.Fatalf("flows length = %d, want 2", len(got))
	}
	first, ok := got[0].(map[string]interface{})
	if !ok {
		t.Fatalf("first flow = %T, want map", got[0])
	}
	if first["id"] != "login" || first["name"] != "Login" || first["enabled"] != true {
		t.Fatalf("first flow = %#v", first)
	}
}

func TestDecodeFlowsRejectsMalformedOrNonListJSON(t *testing.T) {
	t.Parallel()

	for name, input := range map[string]string{
		"malformed": `{`,
		"object":    `{"id":"login"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			diag := &fakeDiagnostics{}
			got, ok := decodeFlows(input, diag)
			if ok {
				t.Fatalf("decode = %#v, want failure", got)
			}
			if len(diag.errors) != 1 {
				t.Fatalf("diagnostics = %#v, want one error", diag.errors)
			}
		})
	}
}

func TestDecodeFlowsAllowsExplicitEmptyList(t *testing.T) {
	t.Parallel()

	diag := &fakeDiagnostics{}

	got, ok := decodeFlows(`[]`, diag)
	if !ok {
		t.Fatalf("expected decode success, got diagnostics %#v", diag.errors)
	}
	if len(got) != 0 {
		t.Fatalf("flows = %#v, want empty", got)
	}
}

func TestEncodeFlowsProducesStableJSON(t *testing.T) {
	t.Parallel()

	got, err := encodeFlows([]interface{}{
		map[string]interface{}{"id": "login", "enabled": true},
	})
	if err != nil {
		t.Fatalf("encode flows: %v", err)
	}
	want := `[{"enabled":true,"id":"login"}]`
	if got != want {
		t.Fatalf("json = %q, want %q", got, want)
	}
}

func TestEncodeFlowsReturnsMarshalError(t *testing.T) {
	t.Parallel()

	_, err := encodeFlows([]interface{}{func() {}})
	if err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestFakeDiagnosticsRecordsSummaryAndDetail(t *testing.T) {
	t.Parallel()

	diag := &fakeDiagnostics{}
	diag.AddError("summary", "detail")

	want := []diagnosticError{{summary: "summary", detail: "detail"}}
	if !reflect.DeepEqual(diag.errors, want) {
		t.Fatalf("errors = %#v, want %#v", diag.errors, want)
	}
}

type fakeDiagnostics struct {
	errors []diagnosticError
}

type diagnosticError struct {
	summary string
	detail  string
}

func (d *fakeDiagnostics) AddError(summary string, detail string) {
	d.errors = append(d.errors, diagnosticError{summary: summary, detail: detail})
}
