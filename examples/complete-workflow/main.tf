# Complete Workflow Example: Creating and querying AppTrust applications
# This example shows a complete workflow of creating applications and querying them

terraform {
  required_providers {
    apptrust = {
      source  = "jfrog/apptrust"
      version = "~> 1.0"
    }
  }
}

# Step 1: Create multiple applications
resource "apptrust_application" "frontend" {
  application_key  = "frontend-app"
  application_name = "Frontend Application"
  project_key      = "web-project"
  description      = "Customer-facing frontend application"
  maturity_level   = "production"
  criticality      = "high"

  labels = {
    tier        = "frontend"
    environment = "production"
    team        = "frontend-team"
  }

  user_owners = ["frontend-lead"]
  group_owners = ["frontend-team"]
}

resource "apptrust_application" "backend" {
  application_key  = "backend-api"
  application_name = "Backend API Service"
  project_key      = "web-project"
  description      = "RESTful API backend service"
  maturity_level   = "production"
  criticality      = "critical"

  labels = {
    tier        = "backend"
    environment = "production"
    team        = "backend-team"
    api-version = "v2"
  }

  user_owners = ["backend-lead"]
  group_owners = ["backend-team", "platform-team"]
}

resource "apptrust_application" "database" {
  application_key  = "database-service"
  application_name = "Database Service"
  project_key      = "web-project"
  description      = "Database management service"
  maturity_level   = "production"
  criticality      = "critical"

  labels = {
    tier        = "data"
    environment = "production"
    team        = "data-team"
  }

  user_owners = ["dba-lead"]
  group_owners = ["data-team"]
}

# Step 2: Query individual applications
data "apptrust_application" "frontend_info" {
  application_key = apptrust_application.frontend.application_key
}

data "apptrust_application" "backend_info" {
  application_key = apptrust_application.backend.application_key
}

# Step 3: Query all applications in the project
data "apptrust_applications" "all_web_apps" {
  project_key = "web-project"
}

# Step 4: Query production applications only
data "apptrust_applications" "production_apps" {
  project_key = "web-project"
  maturity    = "production"
}

# Step 5: Query critical applications
data "apptrust_applications" "critical_apps" {
  project_key = "web-project"
  criticality = "critical"
}

# Step 6: Query by team label
data "apptrust_applications" "frontend_team_apps" {
  project_key = "web-project"
  labels = [
    "team:frontend-team"
  ]
}

# Outputs
output "created_applications" {
  value = {
    frontend = {
      key         = apptrust_application.frontend.application_key
      name        = apptrust_application.frontend.application_name
      maturity    = apptrust_application.frontend.maturity_level
      criticality = apptrust_application.frontend.criticality
    }
    backend = {
      key         = apptrust_application.backend.application_key
      name        = apptrust_application.backend.application_name
      maturity    = apptrust_application.backend.maturity_level
      criticality = apptrust_application.backend.criticality
    }
    database = {
      key         = apptrust_application.database.application_key
      name        = apptrust_application.database.application_name
      maturity    = apptrust_application.database.maturity_level
      criticality = apptrust_application.database.criticality
    }
  }
  description = "Details of all created applications"
}

output "project_statistics" {
  value = {
    total_applications     = data.apptrust_applications.all_web_apps.total
    production_apps_count  = data.apptrust_applications.production_apps.total
    critical_apps_count    = data.apptrust_applications.critical_apps.total
    frontend_team_apps     = data.apptrust_applications.frontend_team_apps.total
  }
  description = "Statistics about applications in the project"
}

output "application_details" {
  value = {
    frontend = {
      key         = data.apptrust_application.frontend_info.application_key
      name        = data.apptrust_application.frontend_info.application_name
      description = data.apptrust_application.frontend_info.description
      labels      = data.apptrust_application.frontend_info.labels
      owners      = concat(
        data.apptrust_application.frontend_info.user_owners,
        data.apptrust_application.frontend_info.group_owners
      )
    }
    backend = {
      key         = data.apptrust_application.backend_info.application_key
      name        = data.apptrust_application.backend_info.application_name
      description = data.apptrust_application.backend_info.description
      labels      = data.apptrust_application.backend_info.labels
      owners      = concat(
        data.apptrust_application.backend_info.user_owners,
        data.apptrust_application.backend_info.group_owners
      )
    }
  }
  description = "Detailed information about queried applications"
}

