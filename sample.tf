terraform {
  required_providers {
    apptrust = {
      source  = "jfrog/apptrust"
      version = "1.0.0"
    }
  }
}

provider "apptrust" {
  url = "https://myinstance.jfrog.io/artifactory"
  // supply JFROG_ACCESS_TOKEN (Identity Token with Admin privileges) as env var
}

