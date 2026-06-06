package usercollections

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSource = &UserFactorsDataSource{}

type UserFactorsDataSource struct {
	collectionDataSource
}

func NewUserFactorsDataSource() datasource.DataSource {
	return &UserFactorsDataSource{
		collectionDataSource: collectionDataSource{
			typeSuffix:  "user_factors",
			collection:  "factors",
			resultName:  "factors",
			description: "Reads enrolled MFA factors for a Gravitee AM user",
		},
	}
}

func (d *UserFactorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_factors"
}
