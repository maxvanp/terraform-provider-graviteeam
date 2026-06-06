package usercollections

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSource = &UserIdentitiesDataSource{}

type UserIdentitiesDataSource struct {
	collectionDataSource
}

func NewUserIdentitiesDataSource() datasource.DataSource {
	return &UserIdentitiesDataSource{
		collectionDataSource: collectionDataSource{
			typeSuffix:  "user_identities",
			collection:  "identities",
			resultName:  "identities",
			description: "Reads linked identities for a Gravitee AM user",
		},
	}
}

func (d *UserIdentitiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_identities"
}
