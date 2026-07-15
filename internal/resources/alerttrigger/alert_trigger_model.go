package alerttrigger

import "github.com/hashicorp/terraform-plugin-framework/types"

type AlertTriggerModel struct {
	ID               types.String `tfsdk:"id"`
	DomainID         types.String `tfsdk:"domain_id"`
	Type             types.String `tfsdk:"type"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	AlertNotifierIDs types.Set    `tfsdk:"alert_notifier_ids"`
}
