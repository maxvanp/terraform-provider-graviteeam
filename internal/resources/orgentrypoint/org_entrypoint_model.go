package orgentrypoint

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgEntrypointModel struct {
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	Description       types.String   `tfsdk:"description"`
	URL               types.String   `tfsdk:"url"`
	Tags              []types.String `tfsdk:"tags"`
	DefaultEntrypoint types.Bool     `tfsdk:"default_entrypoint"`
}
