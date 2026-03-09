package factor

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type FactorModel struct {
	ID         types.String `tfsdk:"id"`
	DomainID   types.String `tfsdk:"domain_id"`
	Name       types.String `tfsdk:"name"`
	FactorType types.String `tfsdk:"factor_type"`
}
