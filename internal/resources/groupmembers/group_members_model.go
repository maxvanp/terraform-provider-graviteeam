package groupmembers

import "github.com/hashicorp/terraform-plugin-framework/types"

type GroupMembersModel struct {
	DomainID types.String   `tfsdk:"domain_id"`
	GroupID  types.String   `tfsdk:"group_id"`
	Members  []types.String `tfsdk:"members"`
}
