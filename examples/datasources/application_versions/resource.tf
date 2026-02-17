data "apptrust_application_versions" "example" {
  application_key = "my-web-app"
}

output "versions" {
  value = data.apptrust_application_versions.example.versions
}

output "total" {
  value = data.apptrust_application_versions.example.total
}
