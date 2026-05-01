package protectedresource

import "github.com/hashicorp/terraform-plugin-framework/types"

type ProtectedResourceModel struct {
	ID                  types.String                    `tfsdk:"id"`
	DomainID            types.String                    `tfsdk:"domain_id"`
	Name                types.String                    `tfsdk:"name"`
	Description         types.String                    `tfsdk:"description"`
	Type                types.String                    `tfsdk:"type"`
	ResourceIdentifiers []types.String                  `tfsdk:"resource_identifiers"`
	SettingsJSON        types.String                    `tfsdk:"settings_json"`
	ClientID            types.String                    `tfsdk:"client_id"`
	ClientSecret        types.String                    `tfsdk:"client_secret"`
	Features            []ProtectedResourceFeatureModel `tfsdk:"feature"`
}

type ProtectedResourceFeatureModel struct {
	Key         types.String   `tfsdk:"key"`
	Type        types.String   `tfsdk:"type"`
	Description types.String   `tfsdk:"description"`
	Scopes      []types.String `tfsdk:"scopes"`
}
