package user

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserModel struct {
	ID                  types.String `tfsdk:"id"`
	DomainID            types.String `tfsdk:"domain_id"`
	Username            types.String `tfsdk:"username"`
	Email               types.String `tfsdk:"email"`
	FirstName           types.String `tfsdk:"first_name"`
	LastName            types.String `tfsdk:"last_name"`
	DisplayName         types.String `tfsdk:"display_name"`
	ForceResetPassword  types.Bool   `tfsdk:"force_reset_password"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	Locked              types.Bool   `tfsdk:"locked"`
	PreRegistration     types.Bool   `tfsdk:"pre_registration"`
	ResetPassword       types.String `tfsdk:"reset_password"`
	ResetTrigger        types.String `tfsdk:"reset_password_trigger"`
	RegistrationTrigger types.String `tfsdk:"registration_confirmation_trigger"`
}
