resource "graviteeam_identity_provider" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "Corporate Directory"
  type      = "inline-am-idp"
  configuration = jsonencode({
    users = [{
      firstname = "John"
      lastname  = "Doe"
      username  = "jdoe@example.com"
      email     = "jdoe@example.com"
      password  = "SecurePass123!"
    }]
  })
  mappers = {
    username = "username"
    email    = "email"
  }
  group_mapper = {
    "{#profile['groups'] != null && #profile['groups'].contains('admin-group')}" = [
      graviteeam_group.example.id
    ]
  }
}
