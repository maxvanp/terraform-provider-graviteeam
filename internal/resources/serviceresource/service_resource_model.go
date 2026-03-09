package serviceresource

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ServiceResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DomainID      types.String `tfsdk:"domain_id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	Configuration types.String `tfsdk:"configuration"`
}
