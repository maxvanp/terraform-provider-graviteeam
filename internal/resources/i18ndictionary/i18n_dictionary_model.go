package i18ndictionary

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type I18nDictionaryModel struct {
	ID       types.String `tfsdk:"id"`
	DomainID types.String `tfsdk:"domain_id"`
	Name     types.String `tfsdk:"name"`
	Locale   types.String `tfsdk:"locale"`
	Entries  types.Map    `tfsdk:"entries"`
}
