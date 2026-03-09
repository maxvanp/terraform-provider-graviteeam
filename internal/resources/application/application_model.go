package application

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ApplicationModel struct {
	ID                    types.String                `tfsdk:"id"`
	DomainID              types.String                `tfsdk:"domain_id"`
	Name                  types.String                `tfsdk:"name"`
	Type                  types.String                `tfsdk:"type"`
	Description           types.String                `tfsdk:"description"`
	ClientID              types.String                `tfsdk:"client_id"`
	ClientSecret          types.String                `tfsdk:"client_secret"`
	IdentityProviders     []types.String              `tfsdk:"identity_providers"`
	IdentityProviderRules []IdentityProviderRuleModel `tfsdk:"identity_provider_rule"`
	Factors               []types.String              `tfsdk:"factors"`
	OAuthSettings         *OAuthSettingsModel         `tfsdk:"oauth_settings"`
	MFASettings           *MFASettingsModel           `tfsdk:"mfa_settings"`
}

type IdentityProviderRuleModel struct {
	Identity      types.String `tfsdk:"identity"`
	SelectionRule types.String `tfsdk:"selection_rule"`
	Priority      types.Int64  `tfsdk:"priority"`
}

type OAuthSettingsModel struct {
	RedirectURIs                []types.String `tfsdk:"redirect_uris"`
	PostLogoutRedirectURIs      []types.String `tfsdk:"post_logout_redirect_uris"`
	GrantTypes                  []types.String `tfsdk:"grant_types"`
	ResponseTypes               []types.String `tfsdk:"response_types"`
	Scopes                      []types.String `tfsdk:"scopes"`
	AccessTokenValiditySeconds  types.Int64    `tfsdk:"access_token_validity_seconds"`
	RefreshTokenValiditySeconds types.Int64    `tfsdk:"refresh_token_validity_seconds"`
	IDTokenValiditySeconds      types.Int64    `tfsdk:"id_token_validity_seconds"`
}

type MFASettingsModel struct {
	Enrollment types.String `tfsdk:"enrollment"`
	Challenge  types.String `tfsdk:"challenge"`
}
