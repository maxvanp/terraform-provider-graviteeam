package form

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type FormModel struct {
	ID            types.String `tfsdk:"id"`
	DomainID      types.String `tfsdk:"domain_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	Template      types.String `tfsdk:"template"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Content       types.String `tfsdk:"content"`
}
