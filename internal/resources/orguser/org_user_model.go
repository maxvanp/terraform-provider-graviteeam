package orguser

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgUserModel struct {
	ID              types.String `tfsdk:"id"`
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password"`
	Email           types.String `tfsdk:"email"`
	FirstName       types.String `tfsdk:"first_name"`
	LastName        types.String `tfsdk:"last_name"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	PreRegistration types.Bool   `tfsdk:"pre_registration"`
	ResetPassword   types.String `tfsdk:"reset_password"`
	ResetTrigger    types.String `tfsdk:"reset_password_trigger"`
}
