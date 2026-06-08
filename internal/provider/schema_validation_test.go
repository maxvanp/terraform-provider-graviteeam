package provider

import (
	"context"
	"sort"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestProviderMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("1.2.3-test")()

	resp := &fwprovider.MetadataResponse{}
	p.Metadata(ctx, fwprovider.MetadataRequest{}, resp)

	if got, want := resp.TypeName, "graviteeam"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
	if got, want := resp.Version, "1.2.3-test"; got != want {
		t.Fatalf("version = %q, want %q", got, want)
	}
}

func TestProviderSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()

	req := fwprovider.SchemaRequest{}
	resp := &fwprovider.SchemaResponse{}
	p.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema diagnostics: %+v", resp.Diagnostics)
	}

	diagnostics := resp.Schema.ValidateImplementation(ctx)
	if diagnostics.HasError() {
		t.Fatalf("provider schema validation diagnostics: %+v", diagnostics)
	}
}

func TestProviderSchemaAttributes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()

	resp := &fwprovider.SchemaResponse{}
	p.Schema(ctx, fwprovider.SchemaRequest{}, resp)

	assertProviderStringAttribute(t, resp.Schema.Attributes, "api_url", true, false, false)
	assertProviderStringAttribute(t, resp.Schema.Attributes, "client_id", true, false, false)
	assertProviderStringAttribute(t, resp.Schema.Attributes, "client_secret", true, false, true)
	assertProviderStringAttribute(t, resp.Schema.Attributes, "organization_id", false, true, false)
	assertProviderStringAttribute(t, resp.Schema.Attributes, "environment_id", false, true, false)
}

func TestProviderConfigureBuildsClientWithDefaultScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()
	schemaResp := &fwprovider.SchemaResponse{}
	p.Schema(ctx, fwprovider.SchemaRequest{}, schemaResp)
	resp := &fwprovider.ConfigureResponse{}

	p.Configure(ctx, fwprovider.ConfigureRequest{
		Config: providerConfig(schemaResp.Schema, "http://example.test", "client-id", "client-secret", "", ""),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %+v", resp.Diagnostics)
	}
	got, ok := resp.ResourceData.(*client.Client)
	if !ok {
		t.Fatalf("resource data = %T, want *client.Client", resp.ResourceData)
	}
	if resp.DataSourceData != got {
		t.Fatalf("data source data should reuse resource client")
	}
	if got.BaseURL != "http://example.test" || got.OrganizationID != "DEFAULT" || got.EnvironmentID != "DEFAULT" {
		t.Fatalf("client = %#v, want default organization/environment scope", got)
	}
}

func TestProviderConfigureBuildsClientWithConfiguredScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()
	schemaResp := &fwprovider.SchemaResponse{}
	p.Schema(ctx, fwprovider.SchemaRequest{}, schemaResp)
	resp := &fwprovider.ConfigureResponse{}

	p.Configure(ctx, fwprovider.ConfigureRequest{
		Config: providerConfig(schemaResp.Schema, "http://example.test", "client-id", "client-secret", "ORG", "ENV"),
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %+v", resp.Diagnostics)
	}
	got, ok := resp.ResourceData.(*client.Client)
	if !ok {
		t.Fatalf("resource data = %T, want *client.Client", resp.ResourceData)
	}
	if got.OrganizationID != "ORG" || got.EnvironmentID != "ENV" {
		t.Fatalf("client scope = %s/%s, want ORG/ENV", got.OrganizationID, got.EnvironmentID)
	}
}

func TestProviderConfigureStopsOnInvalidConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()
	schemaResp := &fwprovider.SchemaResponse{}
	p.Schema(ctx, fwprovider.SchemaRequest{}, schemaResp)
	resp := &fwprovider.ConfigureResponse{}

	p.Configure(ctx, fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Raw:    tftypes.NewValue(tftypes.String, "not an object"),
			Schema: schemaResp.Schema,
		},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostics")
	}
	if resp.ResourceData != nil || resp.DataSourceData != nil {
		t.Fatalf("expected no configured client, got resource=%T datasource=%T", resp.ResourceData, resp.DataSourceData)
	}
}

func TestProviderRegistersExpectedResourceTypeNames(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()

	got := registeredResourceTypeNames(ctx, p.Resources(ctx))
	want := []string{
		"graviteeam_alert_notifier",
		"graviteeam_alert_trigger",
		"graviteeam_application",
		"graviteeam_application_email",
		"graviteeam_application_flow",
		"graviteeam_application_form",
		"graviteeam_application_member",
		"graviteeam_application_secret",
		"graviteeam_auth_device_notifier",
		"graviteeam_authorization_engine",
		"graviteeam_bot_detection",
		"graviteeam_certificate",
		"graviteeam_device_identifier",
		"graviteeam_domain",
		"graviteeam_domain_certificate_settings",
		"graviteeam_domain_flow",
		"graviteeam_domain_member",
		"graviteeam_email_template",
		"graviteeam_extension_grant",
		"graviteeam_factor",
		"graviteeam_form",
		"graviteeam_generated_certificate",
		"graviteeam_group",
		"graviteeam_group_members",
		"graviteeam_group_roles",
		"graviteeam_i18n_dictionary",
		"graviteeam_identity_provider",
		"graviteeam_identity_provider_password_policy",
		"graviteeam_org_entrypoint",
		"graviteeam_org_form",
		"graviteeam_org_group",
		"graviteeam_org_group_members",
		"graviteeam_org_identity_provider",
		"graviteeam_org_member",
		"graviteeam_org_reporter",
		"graviteeam_org_role",
		"graviteeam_org_settings",
		"graviteeam_org_tag",
		"graviteeam_org_user",
		"graviteeam_org_user_token",
		"graviteeam_password_policy",
		"graviteeam_protected_resource",
		"graviteeam_protected_resource_member",
		"graviteeam_protected_resource_secret",
		"graviteeam_reporter",
		"graviteeam_role",
		"graviteeam_scope",
		"graviteeam_service_resource",
		"graviteeam_theme",
		"graviteeam_user",
		"graviteeam_user_certificate_credential",
		"graviteeam_user_role",
	}
	sort.Strings(want)

	assertStringSlicesEqual(t, got, want)
}

func TestProviderRegistersExpectedDataSourceTypeNames(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()

	got := registeredDataSourceTypeNames(ctx, p.DataSources(ctx))
	want := []string{
		"graviteeam_admin_metadata",
		"graviteeam_analytics",
		"graviteeam_application_metadata",
		"graviteeam_audits",
		"graviteeam_domain_metadata",
		"graviteeam_entrypoints",
		"graviteeam_environment_metadata",
		"graviteeam_flows",
		"graviteeam_form_preview",
		"graviteeam_password_policy_evaluation",
		"graviteeam_permissions_metadata",
		"graviteeam_platform_metadata",
		"graviteeam_plugins",
		"graviteeam_self_metadata",
		"graviteeam_user_consents",
		"graviteeam_user_credentials",
		"graviteeam_user_devices",
		"graviteeam_user_factors",
		"graviteeam_user_identities",
	}
	sort.Strings(want)

	assertStringSlicesEqual(t, got, want)
}

func providerConfig(configSchema schema.Schema, apiURL, clientID, clientSecret, orgID, envID string) tfsdk.Config {
	values := map[string]tftypes.Value{
		"api_url":         tftypes.NewValue(tftypes.String, apiURL),
		"client_id":       tftypes.NewValue(tftypes.String, clientID),
		"client_secret":   tftypes.NewValue(tftypes.String, clientSecret),
		"organization_id": providerConfigOptionalString(orgID),
		"environment_id":  providerConfigOptionalString(envID),
	}
	return tfsdk.Config{
		Raw: tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"api_url":         tftypes.String,
				"client_id":       tftypes.String,
				"client_secret":   tftypes.String,
				"organization_id": tftypes.String,
				"environment_id":  tftypes.String,
			}},
			values,
		),
		Schema: configSchema,
	}
}

func providerConfigOptionalString(value string) tftypes.Value {
	if value == "" {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value)
}

func TestResourceSchemas(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()

	for _, constructor := range p.Resources(ctx) {
		r := constructor()
		req := fwresource.SchemaRequest{}
		resp := &fwresource.SchemaResponse{}
		r.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("%T schema diagnostics: %+v", r, resp.Diagnostics)
		}

		diagnostics := resp.Schema.ValidateImplementation(ctx)
		if diagnostics.HasError() {
			t.Fatalf("%T schema validation diagnostics: %+v", r, diagnostics)
		}
	}
}

func TestDataSourceSchemas(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()

	for _, constructor := range p.DataSources(ctx) {
		d := constructor()
		req := fwdatasource.SchemaRequest{}
		resp := &fwdatasource.SchemaResponse{}
		d.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("%T schema diagnostics: %+v", d, resp.Diagnostics)
		}

		diagnostics := resp.Schema.ValidateImplementation(ctx)
		if diagnostics.HasError() {
			t.Fatalf("%T schema validation diagnostics: %+v", d, diagnostics)
		}
	}
}

func registeredResourceTypeNames(ctx context.Context, constructors []func() fwresource.Resource) []string {
	names := make([]string, 0, len(constructors))
	for _, constructor := range constructors {
		resp := &fwresource.MetadataResponse{}
		constructor().Metadata(ctx, fwresource.MetadataRequest{ProviderTypeName: "graviteeam"}, resp)
		names = append(names, resp.TypeName)
	}
	sort.Strings(names)
	return names
}

func registeredDataSourceTypeNames(ctx context.Context, constructors []func() fwdatasource.DataSource) []string {
	names := make([]string, 0, len(constructors))
	for _, constructor := range constructors {
		resp := &fwdatasource.MetadataResponse{}
		constructor().Metadata(ctx, fwdatasource.MetadataRequest{ProviderTypeName: "graviteeam"}, resp)
		names = append(names, resp.TypeName)
	}
	sort.Strings(names)
	return names
}

func assertProviderStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, sensitive bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Sensitive != sensitive {
		t.Fatalf("%s flags = required:%t optional:%t sensitive:%t, want required:%t optional:%t sensitive:%t",
			name, attr.Required, attr.Optional, attr.Sensitive, required, optional, sensitive)
	}
}

func assertStringSlicesEqual(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("length = %d, want %d\ngot:  %#v\nwant: %#v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names differ at %d: got %q, want %q\ngot:  %#v\nwant: %#v", i, got[i], want[i], got, want)
		}
	}
}
