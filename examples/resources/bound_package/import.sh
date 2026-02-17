#!/usr/bin/env bash
# Usage: ./import.sh <application_key> <package_type> <package_name> <package_version>
# Example: ./import.sh my-web-app maven com.example:my-library 1.2.3
# Import ID format: application_key:package_type:package_name:package_version
terraform import apptrust_bound_package.example "${1}:${2}:${3}:${4}"
