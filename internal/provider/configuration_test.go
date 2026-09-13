package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderConfigurationValidation(t *testing.T) {
	t.Parallel()
	for _, attr := range []string{"api_url", "client_id", "client_secret", "organization_id", "environment_id"} {
		for _, value := range []any{"", " \t", tftypes.UnknownValue} {
			t.Run(attr+"/"+fmt.Sprint(value), func(t *testing.T) {
				ctx := context.Background()
				p := New("test")()
				schemaResp := &fwprovider.SchemaResponse{}
				p.Schema(ctx, fwprovider.SchemaRequest{}, schemaResp)
				config := providerConfig(schemaResp.Schema, "https://am.example.test", "client-id", "secret-not-for-diagnostics", "", "")
				var values map[string]tftypes.Value
				if err := config.Raw.As(&values); err != nil {
					t.Fatal(err)
				}
				values[attr] = tftypes.NewValue(tftypes.String, value)
				config.Raw = tftypes.NewValue(config.Raw.Type(), values)

				configured := &fwprovider.ConfigureResponse{}
				p.Configure(ctx, fwprovider.ConfigureRequest{Config: config}, configured)
				if !configured.Diagnostics.HasError() || configured.ResourceData != nil || configured.DataSourceData != nil {
					t.Fatalf("invalid configuration must return diagnostics without a client: %+v", configured)
				}
				assertConfigurationDiagnostic(t, configured.Diagnostics, attr)

				validator, ok := p.(fwprovider.ProviderWithValidateConfig)
				if !ok {
					t.Fatal("provider must validate configuration offline")
				}
				validated := &fwprovider.ValidateConfigResponse{}
				validator.ValidateConfig(ctx, fwprovider.ValidateConfigRequest{Config: config}, validated)
				if value == tftypes.UnknownValue {
					if validated.Diagnostics.HasError() {
						t.Fatalf("unknown values must be allowed during validation: %+v", validated.Diagnostics)
					}
				} else {
					assertConfigurationDiagnostic(t, validated.Diagnostics, attr)
				}
			})
		}
	}
}

func TestProviderConfigurationURL(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		url   string
		valid bool
	}{
		{"https://am.example.test/management", true},
		{"http://localhost:8093/", true},
		{"http://[::1]:8093", true},
		{"am.example.test", false},
		{"ftp://am.example.test", false},
		{"https:///management", false},
		{"https://am.example.test/%xx", false},
		{"https://user:secret-not-for-diagnostics@am.example.test", false},
		{"https://am.example.test?token=secret-not-for-diagnostics", false},
		{"https://am.example.test#fragment", false},
	} {
		t.Run(tc.url, func(t *testing.T) {
			ctx := context.Background()
			p := New("test")()
			schemaResp := &fwprovider.SchemaResponse{}
			p.Schema(ctx, fwprovider.SchemaRequest{}, schemaResp)
			config := providerConfig(schemaResp.Schema, tc.url, "client-id", "secret-not-for-diagnostics", "", "")
			resp := &fwprovider.ConfigureResponse{}
			p.Configure(ctx, fwprovider.ConfigureRequest{Config: config}, resp)
			if resp.Diagnostics.HasError() == tc.valid {
				t.Fatalf("valid=%t, diagnostics=%+v", tc.valid, resp.Diagnostics)
			}
			if !tc.valid {
				assertConfigurationDiagnostic(t, resp.Diagnostics, "api_url")
				if resp.ResourceData != nil || resp.DataSourceData != nil {
					t.Fatal("invalid URL must not configure a client")
				}
			}
			validated := &fwprovider.ValidateConfigResponse{}
			p.(fwprovider.ProviderWithValidateConfig).ValidateConfig(ctx, fwprovider.ValidateConfigRequest{Config: config}, validated)
			if validated.Diagnostics.HasError() == tc.valid {
				t.Fatalf("offline validation: valid=%t, diagnostics=%+v", tc.valid, validated.Diagnostics)
			}
		})
	}
}

func assertConfigurationDiagnostic(t *testing.T, diagnostics diag.Diagnostics, attr string) {
	t.Helper()
	if !diagnostics.HasError() {
		t.Fatal("expected an error diagnostic")
	}
	for _, diagnostic := range diagnostics {
		withPath, ok := diagnostic.(diag.DiagnosticWithPath)
		if !ok || !withPath.Path().Equal(path.Root(attr)) {
			t.Fatalf("expected diagnostic for %s: %+v", attr, diagnostic)
		}
		if strings.Contains(diagnostic.Detail(), "secret-not-for-diagnostics") {
			t.Fatal("diagnostic exposed a credential")
		}
	}
}
