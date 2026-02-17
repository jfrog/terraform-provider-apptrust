data "apptrust_application" "example" {
  application_key = "my-web-app"
}

output "application_name" {
  value = data.apptrust_application.example.application_name
}

output "project_key" {
  value = data.apptrust_application.example.project_key
}
