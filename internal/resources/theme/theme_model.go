package theme

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ThemeModel struct {
	ID                      types.String `tfsdk:"id"`
	DomainID                types.String `tfsdk:"domain_id"`
	LogoURL                 types.String `tfsdk:"logo_url"`
	LogoWidth               types.Int64  `tfsdk:"logo_width"`
	FaviconURL              types.String `tfsdk:"favicon_url"`
	PrimaryButtonColorHex   types.String `tfsdk:"primary_button_color_hex"`
	SecondaryButtonColorHex types.String `tfsdk:"secondary_button_color_hex"`
	PrimaryTextColorHex     types.String `tfsdk:"primary_text_color_hex"`
	SecondaryTextColorHex   types.String `tfsdk:"secondary_text_color_hex"`
	CSS                     types.String `tfsdk:"css"`
}
