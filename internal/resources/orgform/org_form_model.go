package orgform

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgFormModel struct {
	ID       types.String `tfsdk:"id"`
	Template types.String `tfsdk:"template"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Content  types.String `tfsdk:"content"`
}
