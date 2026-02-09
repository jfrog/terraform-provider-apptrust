# List applications with filters (GET /v1/applications).
# Use maturity (not maturity_level) for the filter. Labels in "key:value" format.
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

  order_by  = "name"   # or "created" (API default)
  order_asc = true
  limit     = 10
  offset    = 0
}

output "filtered_applications" {
  value = {
    total        = data.apptrust_applications.filtered.total
    applications = data.apptrust_applications.filtered.applications
  }
}
