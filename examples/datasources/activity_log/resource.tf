data "apptrust_activity_log" "example" {
  project_key = ["my-project"]
  result      = ["success"]
  sort_by     = "timestamp"
  sort        = "desc"
  limit       = 50
  offset      = 0
}

output "activity_logs" {
  value = data.apptrust_activity_log.example.activity_logs
}

output "total" {
  value = data.apptrust_activity_log.example.total
}
