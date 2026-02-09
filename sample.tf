terraform {
  required_providers {
    apptrust = {
      source  = "jfrog/apptrust",
    }
  }
}

provider "apptrust" {
  url          = "https://myinstance.jfrog.io/artifactory"
  access_token = "" # Set to a valid token, or use JFROG_ACCESS_TOKEN / ARTIFACTORY_ACCESS_TOKEN env var
}

# Example: Create an AppTrust Application
resource "apptrust_application" "example" {
  application_key  = "my-web-app"
  application_name = "My Web Application"
  project_key      = "tp2"
  description      = "A sample web application managed by Terraform"
  maturity_level   = "production"
  criticality = "high"

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

# Example: Create another application with minimal configuration
resource "apptrust_application" "minimal" {
  application_key  = "minimal-app"
  application_name = "Minimal Application"
  project_key      = "tp2"
}

# Example: Read a single application by key
data "apptrust_application" "example" {
  application_key = apptrust_application.example.application_key
}

# Example: List applications with filters
data "apptrust_applications" "filtered" {
  project_key = "tp2"
  maturity    = "production"
  criticality = "high"

  labels = [
    "environment:production",
    "region:us-east"
  ]

  owners = [
    "admin",
    "devops-team"
  ]

  order_by  = "name"
  order_asc = true
  limit     = 10
  offset    = 0
}

# Example: List all applications in a project
data "apptrust_applications" "all" {
  project_key = "tp2"
}

# Output examples
output "application_details" {
  description = "Details of the created application"
  value = {
    key         = apptrust_application.example.application_key
    name        = apptrust_application.example.application_name
    project     = apptrust_application.example.project_key
    maturity    = apptrust_application.example.maturity_level
    criticality = apptrust_application.example.criticality
  }
}

output "application_from_data_source" {
  description = "Application details from data source"
  value = {
    key          = data.apptrust_application.example.application_key
    name         = data.apptrust_application.example.application_name
    project      = data.apptrust_application.example.project_key
    maturity     = data.apptrust_application.example.maturity_level
    criticality  = data.apptrust_application.example.criticality
    labels       = data.apptrust_application.example.labels
    user_owners  = data.apptrust_application.example.user_owners
    group_owners = data.apptrust_application.example.group_owners
  }
}

output "filtered_applications" {
  description = "List of filtered applications"
  value = {
    total        = data.apptrust_applications.filtered.total
    applications = data.apptrust_applications.filtered.applications
  }
}

output "all_applications_count" {
  description = "Total number of applications in the project"
  value       = data.apptrust_applications.all.total
}

