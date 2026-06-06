package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/datasources/analytics"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/datasources/audits"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/datasources/entrypoints"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/datasources/flows"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/alertnotifier"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/alerttrigger"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/application"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/applicationemail"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/applicationflow"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/applicationform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/applicationmember"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/applicationsecret"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/authdevicenotifier"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/authorizationengine"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/botdetection"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/certificate"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/deviceidentifier"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/domain"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/domaincertificatesettings"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/domainflow"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/domainmember"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/emailtemplate"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/extensiongrant"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/factor"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/form"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/group"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/groupmembers"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/grouproles"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/i18ndictionary"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/identityprovider"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/identityproviderpasswordpolicy"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orgentrypoint"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orgform"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orggroup"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orggroupmembers"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orgidentityprovider"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orgmember"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orgreporter"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orgrole"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orgsettings"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orgtag"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orguser"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/orgusertoken"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/passwordpolicy"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/protectedresource"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/protectedresourcemember"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/protectedresourcesecret"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/reporter"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/role"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/scope"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/serviceresource"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/theme"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/user"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/usercertificatecredential"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/userrole"
)

var _ provider.Provider = &GraviteeAMProvider{}

type GraviteeAMProvider struct {
	version string
}

type GraviteeAMProviderModel struct {
	APIURL         types.String `tfsdk:"api_url"`
	ClientID       types.String `tfsdk:"client_id"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GraviteeAMProvider{
			version: version,
		}
	}
}

func (p *GraviteeAMProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "graviteeam"
	resp.Version = p.version
}

func (p *GraviteeAMProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Gravitee Access Management",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Description: "The URL of the Gravitee AM Management API (e.g. http://localhost:8093)",
				Required:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "Client ID for OAuth2 authentication",
				Required:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "Client secret for OAuth2 authentication",
				Required:    true,
				Sensitive:   true,
			},
			"organization_id": schema.StringAttribute{
				Description: "Organization ID (default: DEFAULT)",
				Optional:    true,
			},
			"environment_id": schema.StringAttribute{
				Description: "Environment ID (default: DEFAULT)",
				Optional:    true,
			},
		},
	}
}

func (p *GraviteeAMProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config GraviteeAMProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiURL := config.APIURL.ValueString()
	clientID := config.ClientID.ValueString()
	clientSecret := config.ClientSecret.ValueString()

	orgID := "DEFAULT"
	if !config.OrganizationID.IsNull() && !config.OrganizationID.IsUnknown() {
		orgID = config.OrganizationID.ValueString()
	}

	envID := "DEFAULT"
	if !config.EnvironmentID.IsNull() && !config.EnvironmentID.IsUnknown() {
		envID = config.EnvironmentID.ValueString()
	}

	c := client.New(apiURL, clientID, clientSecret, orgID, envID)

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *GraviteeAMProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		alertnotifier.NewAlertNotifierResource,
		alerttrigger.NewAlertTriggerResource,
		application.NewApplicationResource,
		applicationemail.NewApplicationEmailResource,
		applicationflow.NewApplicationFlowResource,
		applicationform.NewApplicationFormResource,
		applicationmember.NewApplicationMemberResource,
		applicationsecret.NewApplicationSecretResource,
		authorizationengine.NewAuthorizationEngineResource,
		authdevicenotifier.NewAuthDeviceNotifierResource,
		botdetection.NewBotDetectionResource,
		certificate.NewCertificateResource,
		deviceidentifier.NewDeviceIdentifierResource,
		domaincertificatesettings.NewDomainCertificateSettingsResource,
		domain.NewDomainResource,
		domainflow.NewDomainFlowResource,
		domainmember.NewDomainMemberResource,
		emailtemplate.NewEmailTemplateResource,
		extensiongrant.NewExtensionGrantResource,
		factor.NewFactorResource,
		form.NewFormResource,
		group.NewGroupResource,
		groupmembers.NewGroupMembersResource,
		grouproles.NewGroupRolesResource,
		i18ndictionary.NewI18nDictionaryResource,
		identityprovider.NewIdentityProviderResource,
		identityproviderpasswordpolicy.NewIdentityProviderPasswordPolicyResource,
		orgentrypoint.NewOrgEntrypointResource,
		orgform.NewOrgFormResource,
		orggroup.NewOrgGroupResource,
		orggroupmembers.NewOrgGroupMembersResource,
		orgidentityprovider.NewOrgIdentityProviderResource,
		orgmember.NewOrgMemberResource,
		orgreporter.NewOrgReporterResource,
		orgrole.NewOrgRoleResource,
		orgsettings.NewOrgSettingsResource,
		orgtag.NewOrgTagResource,
		orguser.NewOrgUserResource,
		orgusertoken.NewOrgUserTokenResource,
		passwordpolicy.NewPasswordPolicyResource,
		protectedresource.NewProtectedResourceResource,
		protectedresourcemember.NewProtectedResourceMemberResource,
		protectedresourcesecret.NewProtectedResourceSecretResource,
		reporter.NewReporterResource,
		role.NewRoleResource,
		scope.NewScopeResource,
		serviceresource.NewServiceResourceResource,
		theme.NewThemeResource,
		user.NewUserResource,
		usercertificatecredential.NewUserCertificateCredentialResource,
		userrole.NewUserRoleResource,
	}
}

func (p *GraviteeAMProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		analytics.NewAnalyticsDataSource,
		audits.NewAuditsDataSource,
		entrypoints.NewEntrypointsDataSource,
		flows.NewFlowsDataSource,
	}
}
