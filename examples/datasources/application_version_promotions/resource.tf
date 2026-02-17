data "apptrust_application_version_promotions" "example" {
  application_key = "my-web-app"
  version         = "1.0.0"
}

output "promotions" {
  value = data.apptrust_application_version_promotions.example.promotions
}

output "total" {
  value = data.apptrust_application_version_promotions.example.total
}
