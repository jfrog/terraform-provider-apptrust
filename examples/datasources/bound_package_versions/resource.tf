data "apptrust_bound_package_versions" "example" {
  application_key = "my-web-app"
  package_type    = "maven"
  package_name    = "com.example:my-library"
}

output "versions" {
  value = data.apptrust_bound_package_versions.example.versions
}
