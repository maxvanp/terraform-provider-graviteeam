package extensiongrant

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ExtensionGrantModel struct {
	ID               types.String `tfsdk:"id"`
	DomainID         types.String `tfsdk:"domain_id"`
	Name             types.String `tfsdk:"name"`
	Type             types.String `tfsdk:"type"`
	GrantType        types.String `tfsdk:"grant_type"`
	Configuration    types.String `tfsdk:"configuration"`
	IdentityProvider types.String `tfsdk:"identity_provider"`
	CreateUser       types.Bool   `tfsdk:"create_user"`
	UserExists       types.Bool   `tfsdk:"user_exists"`
}
