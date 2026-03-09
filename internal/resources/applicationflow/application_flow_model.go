package applicationflow

import "github.com/hashicorp/terraform-plugin-framework/types"

type ApplicationFlowModel struct {
	DomainID      types.String `tfsdk:"domain_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	Flows         types.String `tfsdk:"flows"`
}
