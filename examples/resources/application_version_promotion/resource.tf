# Promote an application version to a target stage.
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

resource "apptrust_application_version_promotion" "example" {
  application_key = apptrust_application_version.example.application_key
  version         = apptrust_application_version.example.version
  target_stage    = "QA"
  promotion_type  = "copy"
}
