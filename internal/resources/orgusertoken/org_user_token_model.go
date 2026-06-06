package orgusertoken

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgUserTokenModel struct {
	ID      types.String `tfsdk:"id"`
	UserID  types.String `tfsdk:"user_id"`
	TokenID types.String `tfsdk:"token_id"`
	Name    types.String `tfsdk:"name"`
	Token   types.String `tfsdk:"token"`
}
