# =============================================================================
# AppTrust Provider - Sample configuration testing all resources and data sources
# Order: Resources first (by dependency), then Data sources (aligned to provider order)
# =============================================================================
# Prerequisites: Set provider url and access_token (or JFROG_ACCESS_TOKEN).
# Some resources (application_version with source_artifacts, promotion, release,
# rollback, bound_package) require real Artifactory/AppTrust data; adjust or
# comment out as needed for your environment.
# =============================================================================

terraform {
  required_providers {
    apptrust = {
      source = "jfrog/apptrust"
    }
  }
}

provider "apptrust" {
  url          = "https://myinstance.jfrog.io/artifactory"
  access_token = "" # Set to a valid token, or use JFROG_ACCESS_TOKEN / ARTIFACTORY_ACCESS_TOKEN env var
}

# -----------------------------------------------------------------------------
# RESOURCES (in dependency order)
# -----------------------------------------------------------------------------

# 1. Application (base resource)
resource "apptrust_application" "example" {
  application_key  = "my-web-app"
  application_name = "My Web Application"
  project_key      = "my-project" # set to your project key
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

resource "apptrust_application" "minimal" {
  application_key  = "minimal-app"
  application_name = "Minimal Application"
  project_key      = "my-project"
}

# 2. Application version (requires application; at least one source required)
# Replace source_artifacts path with a real artifact path in your repo if create fails
resource "apptrust_application_version" "example" {
  application_key = apptrust_application.example.application_key
  version         = "1.0.0"
  tag             = "stable"

  source_artifacts = [
    {
      path = "generic-repo/path/to/artifact.jar"
      # sha256 = "..." # optional
    }
  ]
  # source_versions = [{ application_key = apptrust_application.minimal.application_key, version = "1.0.0" }]
}

# 3. Application version promotion (promote to a stage, e.g. QA)
# Requires lifecycle stages to exist in the project; adjust target_stage to match your setup
resource "apptrust_application_version_promotion" "example" {
  application_key = apptrust_application_version.example.application_key
  version         = apptrust_application_version.example.version
  target_stage    = "QA"
  promotion_type  = "copy"
  # included_repository_keys = ["my-repo"]
  # excluded_repository_keys = []
}

# 4. Application version release (release to PROD)
# Uncomment or use when version is ready to release to PROD
# resource "apptrust_application_version_release" "example" {
#   application_key = apptrust_application_version.example.application_key
#   version         = apptrust_application_version.example.version
#   promotion_type  = "copy"
# }

# 5. Application version rollback (rollback from a stage)
# Uncomment when you need to rollback; from_stage must match a stage the version was promoted to
# resource "apptrust_application_version_rollback" "example" {
#   application_key = apptrust_application_version.example.application_key
#   version         = apptrust_application_version.example.version
#   from_stage      = "QA"
# }

# 6. Bound package (bind a package version to the application)
# Requires a real package in Artifactory; adjust type/name/version to match your repo
# resource "apptrust_bound_package" "example" {
#   application_key  = apptrust_application.example.application_key
#   package_type     = "maven"
#   package_name     = "com.example:my-library"
#   package_version  = "1.0.0"
# }

# -----------------------------------------------------------------------------
# DATA SOURCES (in provider order: application, applications, versions, status, promotions, package_bindings, bound_package_versions)
# -----------------------------------------------------------------------------

# 1. Single application
data "apptrust_application" "example" {
  application_key = apptrust_application.example.application_key
}

# 2. List applications (with optional filters)
data "apptrust_applications" "filtered" {
  project_key = "my-project"
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

data "apptrust_applications" "all" {
  project_key = "my-project"
}

# 3. Application versions list
data "apptrust_application_versions" "example" {
  application_key = apptrust_application.example.application_key
  limit           = 10
  offset          = 0
  order_asc       = false
  # release_status = "pre_release"
  # tag            = "stable"
  # created_by     = "admin"
}

# 4. Application version status (release status for a specific version)
data "apptrust_application_version_status" "example" {
  application_key = apptrust_application_version.example.application_key
  version         = apptrust_application_version.example.version
}

# 5. Application version promotions list
data "apptrust_application_version_promotions" "example" {
  application_key = apptrust_application_version.example.application_key
  version         = apptrust_application_version.example.version
  limit           = 10
  offset          = 0
  order_asc       = false
  # filter_by = "status"
  # order_by  = "created"
}

# 6. Application package bindings (packages bound to the application)
data "apptrust_application_package_bindings" "example" {
  application_key = apptrust_application.example.application_key
  # name = "my-package"
  # type = "maven"
}

# 7. Bound package versions (versions for a given package bound to the application)
# Requires at least one bound package; use same type/name as in apptrust_bound_package if you enable it
# data "apptrust_bound_package_versions" "example" {
#   application_key  = apptrust_application.example.application_key
#   package_type     = "maven"
#   package_name     = "com.example:my-library"
#   package_version  = "1.0.0"  # optional filter
#   offset           = 0
#   limit            = 25
# }

# -----------------------------------------------------------------------------
# OUTPUTS (for all resources and data sources)
# -----------------------------------------------------------------------------

output "application_details" {
  description = "Created application"
  value = {
    key         = apptrust_application.example.application_key
    name        = apptrust_application.example.application_name
    project     = apptrust_application.example.project_key
    maturity    = apptrust_application.example.maturity_level
    criticality = apptrust_application.example.criticality
  }
}

output "application_version_details" {
  description = "Created application version"
  value = {
    id             = apptrust_application_version.example.id
    application_key = apptrust_application_version.example.application_key
    version        = apptrust_application_version.example.version
    tag            = apptrust_application_version.example.tag
    release_status = apptrust_application_version.example.release_status
    current_stage  = apptrust_application_version.example.current_stage
  }
}

output "application_version_promotion_id" {
  description = "Promotion resource ID"
  value       = apptrust_application_version_promotion.example.id
}

output "data_application" {
  description = "Single application from data source"
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

output "data_applications_filtered" {
  description = "Filtered applications list"
  value = {
    total        = data.apptrust_applications.filtered.total
    applications = data.apptrust_applications.filtered.applications
  }
}

output "data_applications_all_count" {
  description = "Total applications in project"
  value       = data.apptrust_applications.all.total
}

output "data_application_versions" {
  description = "Application versions list"
  value = {
    total   = data.apptrust_application_versions.example.total
    versions = data.apptrust_application_versions.example.versions
  }
}

output "data_application_version_status" {
  description = "Version release status"
  value       = data.apptrust_application_version_status.example.version_release_status
}

output "data_application_version_promotions" {
  description = "Version promotions list"
  value = {
    total      = data.apptrust_application_version_promotions.example.total
    promotions = data.apptrust_application_version_promotions.example.promotions
  }
}

output "data_application_package_bindings" {
  description = "Packages bound to application"
  value = {
    packages   = data.apptrust_application_package_bindings.example.packages
    pagination = data.apptrust_application_package_bindings.example.pagination
  }
}

# Uncomment when apptrust_bound_package and data apptrust_bound_package_versions are in use:
# output "data_bound_package_versions" {
#   description = "Bound package versions"
#   value = {
#     total    = data.apptrust_bound_package_versions.example.total
#     versions = data.apptrust_bound_package_versions.example.versions
#   }
# }
