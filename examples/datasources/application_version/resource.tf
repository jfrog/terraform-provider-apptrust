data "apptrust_application_version" "example" {
  application_key = "my-web-app"
  version         = "1.0.0"
}

output "release_status" {
  value = data.apptrust_application_version.example.release_status
}

output "current_stage" {
  value = data.apptrust_application_version.example.current_stage
}
