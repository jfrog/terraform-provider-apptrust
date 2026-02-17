data "apptrust_applications" "example" {
  project_key = "my-project"
  name        = "my-web-app"
  limit       = 10
  offset      = 0
}

output "applications" {
  value = data.apptrust_applications.example.applications
}

output "total" {
  value = data.apptrust_applications.example.total
}
