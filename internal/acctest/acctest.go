package acctest

import (
	"github.com/maxvanp/terraform-provider-graviteeam/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const ProviderConfig = `
provider "graviteeam" {
  api_url       = "http://localhost:8093"
  client_id     = "admin"
  client_secret = "adminadmin"
}
`

var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"graviteeam": providerserver.NewProtocol6WithError(provider.New("test")()),
}
