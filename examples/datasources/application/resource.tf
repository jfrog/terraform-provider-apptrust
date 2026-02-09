# Query a single application by key (GET /v1/applications/{application_key})
data "apptrust_application" "example" {
  application_key = "my-web-app"
}

# Outputs (optional)
output "application_key" {
  value = data.apptrust_application.example.application_key
}

output "application_name" {
  value = data.apptrust_application.example.application_name
}

output "project_key" {
  value = data.apptrust_application.example.project_key
}
