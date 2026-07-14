data "graviteeam_self_metadata" "current_user" {
  kind = "current_user"
}

data "graviteeam_self_metadata" "notifications" {
  kind = "notifications"
}
