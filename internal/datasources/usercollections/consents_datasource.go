package usercollections

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSource = &UserConsentsDataSource{}

type UserConsentsDataSource struct {
	collectionDataSource
}

func NewUserConsentsDataSource() datasource.DataSource {
	return &UserConsentsDataSource{
		collectionDataSource: collectionDataSource{
			typeSuffix:  "user_consents",
			collection:  "consents",
			resultName:  "consents",
			description: "Reads OAuth consent approvals for a Gravitee AM user",
		},
	}
}

func (d *UserConsentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_consents"
}
