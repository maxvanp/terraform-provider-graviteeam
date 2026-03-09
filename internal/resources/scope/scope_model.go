package scope

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ScopeModel struct {
	ID          types.String `tfsdk:"id"`
	DomainID    types.String `tfsdk:"domain_id"`
	Key         types.String `tfsdk:"key"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Discovery   types.Bool   `tfsdk:"discovery"`
	ExpiresIn   types.Int64  `tfsdk:"expires_in"`
}
