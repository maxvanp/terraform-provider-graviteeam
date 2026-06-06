package orggroupmembers

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgGroupMembersModel struct {
	GroupID types.String   `tfsdk:"group_id"`
	Members []types.String `tfsdk:"members"`
}
