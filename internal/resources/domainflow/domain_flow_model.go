package domainflow

import "github.com/hashicorp/terraform-plugin-framework/types"

type DomainFlowModel struct {
	DomainID types.String `tfsdk:"domain_id"`
	Flows    types.String `tfsdk:"flows"`
}
