package usercollections

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSource = &UserCredentialsDataSource{}

type UserCredentialsDataSource struct {
	collectionDataSource
}

func NewUserCredentialsDataSource() datasource.DataSource {
	return &UserCredentialsDataSource{
		collectionDataSource: collectionDataSource{
			typeSuffix:  "user_credentials",
			collection:  "credentials",
			resultName:  "credentials",
			description: "Reads credentials for a Gravitee AM user",
		},
	}
}

func (d *UserCredentialsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_credentials"
}
