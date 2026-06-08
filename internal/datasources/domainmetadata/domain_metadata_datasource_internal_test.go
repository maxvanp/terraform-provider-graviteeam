package domainmetadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp datasource.MetadataResponse
	NewDomainMetadataDataSource().Metadata(context.Background(), datasource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_domain_metadata"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewDomainMetadataDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "kind", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "certificate_id", false, true, false)
	assertStringAttribute(t, resp.Schema.Attributes, "result_json", false, false, true)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&DomainMetadataDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp datasource.ConfigureResponse
	(&DomainMetadataDataSource{}).Configure(context.Background(), datasource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	dataSource := &DomainMetadataDataSource{}
	var resp datasource.ConfigureResponse

	dataSource.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if dataSource.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestFormatJSONHandlesEmptyTextAndJSON(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		raw  []byte
		want string
	}{
		"empty": {nil, "null"},
		"text":  {[]byte("public key"), `"public key"`},
		"json":  {[]byte(`{"id":"policy-1","enabled":true}`), "{\n  \"enabled\": true,\n  \"id\": \"policy-1\"\n}"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := formatJSON(test.raw)
			if got != test.want {
				t.Fatalf("json = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDomainMetadataReadResolvesSupportedKinds(t *testing.T) {
	var paths []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/activePolicy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`{"id":"policy-1"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123/key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`public key`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123/keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`["key-1","key-2"]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &DomainMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []struct {
		name  string
		model DomainMetadataModel
		want  string
	}{
		{
			name: "active password policy",
			model: DomainMetadataModel{
				DomainID: types.StringValue("domain-123"),
				Kind:     types.StringValue("active_password_policy"),
			},
			want: "{\n  \"id\": \"policy-1\"\n}",
		},
		{
			name: "certificate key",
			model: DomainMetadataModel{
				DomainID:      types.StringValue("domain-123"),
				Kind:          types.StringValue("certificate_key"),
				CertificateID: types.StringValue("cert-123"),
			},
			want: `"public key"`,
		},
		{
			name: "certificate keys",
			model: DomainMetadataModel{
				DomainID:      types.StringValue("domain-123"),
				Kind:          types.StringValue("certificate_keys"),
				CertificateID: types.StringValue("cert-123"),
			},
			want: "[\n  \"key-1\",\n  \"key-2\"\n]",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			config := domainMetadataConfig(schemaResp.Schema, tt.model)
			readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

			dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
			}
			var state DomainMetadataModel
			if diags := readResp.State.Get(context.Background(), &state); diags.HasError() {
				t.Fatalf("get state: %#v", diags)
			}
			if state.ResultJSON.ValueString() != tt.want {
				t.Fatalf("result_json = %q, want %q", state.ResultJSON.ValueString(), tt.want)
			}
		})
	}

	wantPaths := []string{
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/activePolicy",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123/key",
		"/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/certificates/cert-123/keys",
	}
	if strings.Join(paths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
}

func TestDomainMetadataReadValidatesRequestShapeBeforeHTTP(t *testing.T) {
	dataSource := &DomainMetadataDataSource{}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cases := []DomainMetadataModel{
		{
			DomainID: types.StringValue("domain-123"),
			Kind:     types.StringValue("unknown"),
		},
		{
			DomainID: types.StringValue("domain-123"),
			Kind:     types.StringValue("certificate_key"),
		},
		{
			DomainID: types.StringValue("domain-123"),
			Kind:     types.StringValue("certificate_keys"),
		},
	}

	for _, model := range cases {
		config := domainMetadataConfig(schemaResp.Schema, model)
		readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
		if !readResp.Diagnostics.HasError() {
			t.Fatalf("expected diagnostics for model %#v", model)
		}
	}
}

func TestDomainMetadataReadReportsRemoteError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/password-policies/activePolicy", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "metadata failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	dataSource := &DomainMetadataDataSource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	config := domainMetadataConfig(schemaResp.Schema, DomainMetadataModel{
		DomainID: types.StringValue("domain-123"),
		Kind:     types.StringValue("active_password_policy"),
	})

	readResp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: config}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected remote error diagnostics")
	}
}

func domainMetadataConfig(schema datasourceschema.Schema, model DomainMetadataModel) tfsdk.Config {
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"domain_id":      tftypes.String,
				"kind":           tftypes.String,
				"certificate_id": tftypes.String,
				"result_json":    tftypes.String,
			}},
			map[string]tftypes.Value{
				"domain_id":      tftypes.NewValue(tftypes.String, model.DomainID.ValueString()),
				"kind":           tftypes.NewValue(tftypes.String, model.Kind.ValueString()),
				"certificate_id": domainMetadataStringConfigValue(model.CertificateID),
				"result_json":    tftypes.NewValue(tftypes.String, nil),
			},
		),
		Schema: schema,
	}
}

func domainMetadataStringConfigValue(value types.String) tftypes.Value {
	if value.IsNull() || value.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value.ValueString())
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}

func TestFormatJSONEscapesInvalidJSONText(t *testing.T) {
	t.Parallel()

	got := formatJSON([]byte("line\nbreak"))
	var decoded string
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("formatted text is not a JSON string: %v", err)
	}
	if decoded != "line\nbreak" {
		t.Fatalf("decoded = %q, want original text", decoded)
	}
}
