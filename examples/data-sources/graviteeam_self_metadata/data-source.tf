data "graviteeam_self_metadata" "current_user" {
  kind = "current_user"
}

data "graviteeam_self_metadata" "newsletter_taglines" {
  kind = "newsletter_taglines"
}

data "graviteeam_self_metadata" "notifications" {
  kind = "notifications"
}
