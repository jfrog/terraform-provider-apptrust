data "apptrust_application_package_bindings" "example" {
  application_key = "my-web-app"
}

output "packages" {
  value = data.apptrust_application_package_bindings.example.packages
}
