package orgreporter

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgReporterModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	Configuration types.String `tfsdk:"configuration"`
	Enabled       types.Bool   `tfsdk:"enabled"`
}
