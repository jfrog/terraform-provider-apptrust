#!/usr/bin/env bash
# Usage: ./import.sh <application_key> <version> <target_stage>
# Example: ./import.sh my-web-app 1.0.0 QA
# Import ID format: application_key:version:target_stage
terraform import apptrust_application_version_promotion.example "${1}:${2}:${3}"
