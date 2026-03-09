package group

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GroupModel struct {
	ID          types.String   `tfsdk:"id"`
	DomainID    types.String   `tfsdk:"domain_id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	Members     []types.String `tfsdk:"members"`
	Roles       []types.String `tfsdk:"roles"`
}
