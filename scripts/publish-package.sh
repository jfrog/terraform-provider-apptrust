#!/bin/bash

# Script to publish a Maven artifact as a package in AppTrust
# Usage: ./publish-package.sh --url <url> --token <token> --group-id <groupId> --artifact-id <artifactId> --version <version> --repository <repo>

set -e

# Default values
ARTIFACTORY_URL=""
ACCESS_TOKEN=""
GROUP_ID=""
ARTIFACT_ID=""
VERSION=""
REPOSITORY=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --url)
      ARTIFACTORY_URL="$2"
      shift 2
      ;;
    --token)
      ACCESS_TOKEN="$2"
      shift 2
      ;;
    --group-id)
      GROUP_ID="$2"
      shift 2
      ;;
    --artifact-id)
      ARTIFACT_ID="$2"
      shift 2
      ;;
    --version)
      VERSION="$2"
      shift 2
      ;;
    --repository)
      REPOSITORY="$2"
      shift 2
      ;;
    --help)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --url <url>           Artifactory URL (e.g., https://your-instance.jfrog.io/artifactory)"
      echo "  --token <token>       Access token"
      echo "  --group-id <groupId>  Maven group ID"
      echo "  --artifact-id <id>    Maven artifact ID"
      echo "  --version <version>   Package version"
      echo "  --repository <repo>   Repository name"
      echo "  --help                Show this help message"
      echo ""
      echo "Environment variables can also be used:"
      echo "  JFROG_URL or ARTIFACTORY_URL"
      echo "  JFROG_ACCESS_TOKEN or ARTIFACTORY_ACCESS_TOKEN"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# Check environment variables if not provided via arguments
if [ -z "$ARTIFACTORY_URL" ]; then
  ARTIFACTORY_URL="${JFROG_URL:-${ARTIFACTORY_URL}}"
fi

if [ -z "$ACCESS_TOKEN" ]; then
  ACCESS_TOKEN="${JFROG_ACCESS_TOKEN:-${ARTIFACTORY_ACCESS_TOKEN}}"
fi

# Validate required parameters
if [ -z "$ARTIFACTORY_URL" ]; then
  echo "Error: Artifactory URL is required"
  echo "Provide via --url or set JFROG_URL/ARTIFACTORY_URL environment variable"
  exit 1
fi

if [ -z "$ACCESS_TOKEN" ]; then
  echo "Error: Access token is required"
  echo "Provide via --token or set JFROG_ACCESS_TOKEN/ARTIFACTORY_ACCESS_TOKEN environment variable"
  exit 1
fi

if [ -z "$GROUP_ID" ] || [ -z "$ARTIFACT_ID" ] || [ -z "$VERSION" ]; then
  echo "Error: Group ID, Artifact ID, and Version are required"
  echo "Provide via --group-id, --artifact-id, and --version"
  exit 1
fi

# Remove trailing slash from URL if present
ARTIFACTORY_URL="${ARTIFACTORY_URL%/}"

# Construct package name
PACKAGE_NAME="${GROUP_ID}:${ARTIFACT_ID}:${VERSION}"

echo "Publishing Maven package to AppTrust..."
echo "URL: $ARTIFACTORY_URL"
echo "Package: $PACKAGE_NAME"
echo "Repository: ${REPOSITORY:-'default'}"

# Prepare JSON payload
PAYLOAD=$(cat <<EOF
{
  "type": "maven",
  "name": "$PACKAGE_NAME",
  "repository": "${REPOSITORY:-maven-repo}",
  "version": "$VERSION",
  "groupId": "$GROUP_ID",
  "artifactId": "$ARTIFACT_ID"
}
EOF
)

# Try different API endpoint variations
ENDPOINTS=(
  "/api/apptrust/v1/packages"
  "/api/apptrust/packages"
  "/api/v1/apptrust/packages"
)

SUCCESS=false
for ENDPOINT in "${ENDPOINTS[@]}"; do
  echo ""
  echo "Trying endpoint: ${ARTIFACTORY_URL}${ENDPOINT}"
  
  RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST \
    "${ARTIFACTORY_URL}${ENDPOINT}" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" 2>&1) || true
  
  HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
  BODY=$(echo "$RESPONSE" | sed '$d')
  
  if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
    echo "✓ Success! Package published successfully"
    echo "Response:"
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
    SUCCESS=true
    break
  elif [ "$HTTP_CODE" -eq 404 ]; then
    echo "✗ Endpoint not found (404), trying next..."
    continue
  else
    echo "✗ Error (HTTP $HTTP_CODE):"
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
    if [ "$HTTP_CODE" -ge 400 ] && [ "$HTTP_CODE" -lt 500 ]; then
      # Client error, don't try other endpoints
      break
    fi
  fi
done

if [ "$SUCCESS" = false ]; then
  echo ""
  echo "Failed to publish package. Please check:"
  echo "1. Your Artifactory URL and access token"
  echo "2. AppTrust is properly licensed and enabled"
  echo "3. The API endpoint exists for your Artifactory version"
  echo "4. You have the necessary permissions"
  exit 1
fi

echo ""
echo "Package published successfully! You can now see it in the AppTrust application resources list."

