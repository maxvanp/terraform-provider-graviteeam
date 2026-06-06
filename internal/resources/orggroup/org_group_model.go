package orggroup

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgGroupModel struct {
	ID          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	Members     []types.String `tfsdk:"members"`
	Roles       []types.String `tfsdk:"roles"`
}
