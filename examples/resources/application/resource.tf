resource "apptrust_application" "example" {
  application_key  = "my-web-app"
  application_name = "My Web Application"
  project_key      = "my-project"
  description      = "A sample web application managed by Terraform"
  maturity_level   = "production"
  criticality      = "high"

  labels = {
    environment = "production"
    region      = "us-east"
    team        = "platform"
  }

  user_owners = [
    "admin",
    "devops-team"
  ]

  group_owners = [
    "developers",
    "platform-admins"
  ]
}
