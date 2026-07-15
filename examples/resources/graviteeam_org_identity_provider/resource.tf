resource "graviteeam_org_group" "example" {
  name        = "Org Admins"
  description = "Organization administrators from the external identity provider"
}

resource "graviteeam_org_role" "example" {
  name            = "Org Admin Role"
  description     = "Role mapped from the external identity provider"
  assignable_type = "ORGANIZATION"
}

resource "graviteeam_org_identity_provider" "example" {
  name = "Org Inline IDP"
  type = "inline-am-idp"
  configuration = jsonencode({
    users = []
  })

  mappers = {
    "email"    = "email"
    "username" = "username"
  }

  group_mapper = {
    "{#profile['groups'] != null && #profile['groups'].contains('org-admins')}" = [
      graviteeam_org_group.example.id
    ]
  }

  role_mapper = {
    "{#profile['groups'] != null && #profile['groups'].contains('org-admin-role')}" = [
      graviteeam_org_role.example.id
    ]
  }
}
