package trustdomain

import "github.com/hashicorp/terraform-plugin-framework/types"

type TrustDomainModel struct {
	ID                     types.String   `tfsdk:"id"`
	DomainID               types.String   `tfsdk:"domain_id"`
	Name                   types.String   `tfsdk:"name"`
	Description            types.String   `tfsdk:"description"`
	BundleSource           types.String   `tfsdk:"bundle_source"`
	JWKsURL                types.String   `tfsdk:"jwks_url"`
	RefreshIntervalSeconds types.Int64    `tfsdk:"refresh_interval_seconds"`
	AllowedAlgorithms      []types.String `tfsdk:"allowed_algorithms"`
	CreatedAt              types.Int64    `tfsdk:"created_at"`
	UpdatedAt              types.Int64    `tfsdk:"updated_at"`
}
