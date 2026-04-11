package orgsettings

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgSettingsModel struct {
	ID         types.String `tfsdk:"id"`
	Identities types.List   `tfsdk:"identities"`
}
