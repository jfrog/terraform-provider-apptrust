#!/usr/bin/env bash
# Usage: ./import.sh <application_key> <version> <from_stage>
# Example: ./import.sh my-web-app 1.0.0 QA
# Import ID format: application_key:version:from_stage
terraform import apptrust_application_version_rollback.example "${1}:${2}:${3}"
