package user

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserModel struct {
	ID              types.String `tfsdk:"id"`
	DomainID        types.String `tfsdk:"domain_id"`
	Username        types.String `tfsdk:"username"`
	Email           types.String `tfsdk:"email"`
	FirstName       types.String `tfsdk:"first_name"`
	LastName        types.String `tfsdk:"last_name"`
	PreRegistration types.Bool   `tfsdk:"pre_registration"`
}
