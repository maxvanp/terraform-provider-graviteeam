resource "graviteeam_i18n_dictionary" "example" {
  domain_id = graviteeam_domain.example.id
  name      = "French"
  locale    = "fr"
  entries = {
    "login.title"    = "Connexion"
    "login.username" = "Identifiant"
    "login.password" = "Mot de passe"
  }
}
