package applicationflow

import (
	"reflect"
	"testing"
)

func TestDecodeFlowsParsesCompleteApplicationFlowList(t *testing.T) {
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

func TestDecodeFlowsRejectsMalformedOrNonListApplicationJSON(t *testing.T) {
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

func TestDecodeFlowsAllowsExplicitEmptyApplicationList(t *testing.T) {
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

func TestEncodeFlowsProducesStableApplicationJSON(t *testing.T) {
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

func TestEncodeFlowsReturnsApplicationMarshalError(t *testing.T) {
	t.Parallel()

	_, err := encodeFlows([]interface{}{func() {}})
	if err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestFakeDiagnosticsRecordsApplicationSummaryAndDetail(t *testing.T) {
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
