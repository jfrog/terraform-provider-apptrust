# Roll back the latest promotion of an application version.
resource "apptrust_application" "example" {
  application_key  = "my-web-app"
  application_name = "My Web Application"
  project_key      = "my-project"
}

resource "apptrust_application_version" "example" {
  application_key   = apptrust_application.example.application_key
  version           = "1.0.0"
  tag               = "stable"
  source_artifacts  = [{ path = "generic-repo/path/to/artifact.jar" }]
}

resource "apptrust_application_version_rollback" "example" {
  application_key = apptrust_application_version.example.application_key
  version         = apptrust_application_version.example.version
  from_stage      = "QA"
}
