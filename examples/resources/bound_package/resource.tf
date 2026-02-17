# Bind a package version to an application.
resource "apptrust_application" "example" {
  application_key  = "my-web-app"
  application_name = "My Web Application"
  project_key      = "my-project"
}

resource "apptrust_bound_package" "example" {
  application_key  = apptrust_application.example.application_key
  package_type     = "maven"
  package_name     = "com.example:my-library"
  package_version  = "1.2.3"
}
