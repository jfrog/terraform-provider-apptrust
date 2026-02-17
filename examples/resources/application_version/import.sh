#!/usr/bin/env bash
# Usage: ./import.sh <application_key> <version>
# Example: ./import.sh my-web-app 1.0.0
# Import ID format: application_key:version
terraform import apptrust_application_version.example "${1}:${2}"
