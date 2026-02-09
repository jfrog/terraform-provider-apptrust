#!/usr/bin/env bash
# Usage: ./import.sh <application_key>
# Example: ./import.sh my-web-app
terraform import apptrust_application.example "$1"
