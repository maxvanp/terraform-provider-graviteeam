package domain

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DomainModel struct {
	ID            types.String        `tfsdk:"id"`
	Name          types.String        `tfsdk:"name"`
	Description   types.String        `tfsdk:"description"`
	Enabled       types.Bool          `tfsdk:"enabled"`
	DefaultIdpID  types.String        `tfsdk:"default_idp_id"`
	OIDC          *OIDCModel          `tfsdk:"oidc"`
	LoginSettings *LoginSettingsModel `tfsdk:"login_settings"`
}

type OIDCModel struct {
	AllowLocalhostRedirectURI        types.Bool `tfsdk:"allow_localhost_redirect_uri"`
	AllowHTTPSchemeRedirectURI       types.Bool `tfsdk:"allow_http_scheme_redirect_uri"`
	AllowWildcardRedirectURI         types.Bool `tfsdk:"allow_wildcard_redirect_uri"`
	DynamicClientRegistrationEnabled types.Bool `tfsdk:"dynamic_client_registration_enabled"`
}

type LoginSettingsModel struct {
	RegisterEnabled        types.Bool `tfsdk:"register_enabled"`
	ForgotPasswordEnabled  types.Bool `tfsdk:"forgot_password_enabled"`
	IdentifierFirstEnabled types.Bool `tfsdk:"identifier_first_enabled"`
}
