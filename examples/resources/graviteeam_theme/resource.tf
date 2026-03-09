resource "graviteeam_theme" "example" {
  domain_id                  = graviteeam_domain.example.id
  primary_button_color_hex   = "#1a73e8"
  secondary_button_color_hex = "#e8f0fe"
  primary_text_color_hex     = "#202124"
  secondary_text_color_hex   = "#5f6368"
}
