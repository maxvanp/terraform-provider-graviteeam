package botdetection

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BotDetectionModel struct {
	ID            types.String `tfsdk:"id"`
	DomainID      types.String `tfsdk:"domain_id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	DetectionType types.String `tfsdk:"detection_type"`
	Configuration types.String `tfsdk:"configuration"`
}
