# Application version: at least one source (artifacts, builds, or source_versions) required.
resource "apptrust_application" "example" {
  application_key  = "my-web-app"
  application_name = "My Web Application"
  project_key      = "my-project"
}

resource "apptrust_application_version" "example" {
  application_key = apptrust_application.example.application_key
  version         = "1.0.0"
  tag             = "stable"

  source_artifacts = [
    {
      path = "generic-repo/path/to/artifact.jar"
    }
  ]
}
