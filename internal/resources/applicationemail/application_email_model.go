package applicationemail

import "github.com/hashicorp/terraform-plugin-framework/types"

type ApplicationEmailModel struct {
	ID            types.String `tfsdk:"id"`
	DomainID      types.String `tfsdk:"domain_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	Template      types.String `tfsdk:"template"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	From          types.String `tfsdk:"from"`
	FromName      types.String `tfsdk:"from_name"`
	Subject       types.String `tfsdk:"subject"`
	Content       types.String `tfsdk:"content"`
	ExpiresAfter  types.Int64  `tfsdk:"expires_after"`
}
