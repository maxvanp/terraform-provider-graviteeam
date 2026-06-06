package usercollections

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var _ datasource.DataSource = &UserDevicesDataSource{}

type UserDevicesDataSource struct {
	collectionDataSource
}

func NewUserDevicesDataSource() datasource.DataSource {
	return &UserDevicesDataSource{
		collectionDataSource: collectionDataSource{
			typeSuffix:  "user_devices",
			collection:  "devices",
			resultName:  "devices",
			description: "Reads registered devices for a Gravitee AM user",
		},
	}
}

func (d *UserDevicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_devices"
}
