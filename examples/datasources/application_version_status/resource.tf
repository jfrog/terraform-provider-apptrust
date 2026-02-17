data "apptrust_application_version_status" "example" {
  application_key = "my-web-app"
  version         = "1.0.0"
}

output "version_release_status" {
  value = data.apptrust_application_version_status.example.version_release_status
}
