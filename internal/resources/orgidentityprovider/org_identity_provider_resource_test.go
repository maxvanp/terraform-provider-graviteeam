package orgidentityprovider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/acctest"
)

func TestAccOrgIdentityProviderResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_identity_provider" "test" {
  name          = "test-acc-org-idp"
  type          = "inline-am-idp"
  configuration = jsonencode({})
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graviteeam_org_identity_provider.test", "id"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "name", "test-acc-org-idp"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "type", "inline-am-idp"),
				),
			},
			{
				ResourceName:            "graviteeam_org_identity_provider.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"configuration"},
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_identity_provider" "test" {
  name          = "test-acc-org-idp-updated"
  type          = "inline-am-idp"
  configuration = jsonencode({})
  domain_whitelist = ["example.com"]
  mappers = {
    "email"    = "email"
    "username" = "username"
  }
  group_mapper = {
    "{#profile['groups'] != null && #profile['groups'].contains('test-org-idp-group')}" = [
      graviteeam_org_group.test.id
    ]
  }
  role_mapper = {
    "{#profile['groups'] != null && #profile['groups'].contains('test-org-idp-role')}" = [
      graviteeam_org_role.test.id
    ]
  }
}

resource "graviteeam_org_group" "test" {
  name        = "Test Org IdP Mapped Group"
  description = "Group mapped by organization identity provider test"
}

resource "graviteeam_org_role" "test" {
  name            = "Test Org IdP Mapped Role"
  description     = "Role mapped by organization identity provider test"
  assignable_type = "ORGANIZATION"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "name", "test-acc-org-idp-updated"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "domain_whitelist.0", "example.com"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "mappers.email", "email"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "mappers.username", "username"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "group_mapper.%", "1"),
					resource.TestCheckResourceAttr("graviteeam_org_identity_provider.test", "role_mapper.%", "1"),
				),
			},
			{
				Config: acctest.ProviderConfig + `
resource "graviteeam_org_identity_provider" "test" {
  name          = "test-acc-org-idp-updated"
  type          = "inline-am-idp"
  configuration = jsonencode({})
}

resource "graviteeam_org_group" "test" {
  name        = "Test Org IdP Mapped Group"
  description = "Group mapped by organization identity provider test"
}

resource "graviteeam_org_role" "test" {
  name            = "Test Org IdP Mapped Role"
  description     = "Role mapped by organization identity provider test"
  assignable_type = "ORGANIZATION"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("graviteeam_org_identity_provider.test", "domain_whitelist.0"),
					resource.TestCheckNoResourceAttr("graviteeam_org_identity_provider.test", "mappers.email"),
					resource.TestCheckNoResourceAttr("graviteeam_org_identity_provider.test", "group_mapper.%"),
					resource.TestCheckNoResourceAttr("graviteeam_org_identity_provider.test", "role_mapper.%"),
				),
			},
		},
	})
}
