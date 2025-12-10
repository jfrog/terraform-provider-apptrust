# Terraform Provider for JFrog AppTrust

## Quick Start

Create a new Terraform file with `apptrust` provider:

### HCL Example

```terraform
# Required for Terraform 1.0 and later
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
```

Initialize Terraform:
```sh
$ terraform init
```

Plan (or Apply):
```sh
$ terraform plan
```

Detailed documentation of resources and attributes will be available on [Terraform Registry](https://registry.terraform.io/providers/jfrog/apptrust/latest/docs).

## Resources and Data Sources

Resources and data sources will be documented here as they are implemented.

## Requirements

- Terraform 1.0+
- Artifactory 7.125.0 or later
- Xray 3.130.5 or later
- Enterprise Plus license with AppTrust entitlements
- Access Token with Admin privileges

## Authentication

The provider supports the following authentication methods:

1. **Access Token** (recommended): Set via `access_token` attribute or `JFROG_ACCESS_TOKEN` or `ARTIFACTORY_ACCESS_TOKEN` environment variable
2. **API Key** (deprecated): Set via `api_key` attribute or `ARTIFACTORY_API_KEY` or `JFROG_API_KEY` environment variable

## API Endpoints

This provider uses the JFrog Artifactory REST API to interact with AppTrust features. Specific endpoints will be documented as resources and data sources are implemented.

## Versioning

In general, this project follows [semver](https://semver.org/) as closely as we can for tagging releases of the package. We've adopted the following versioning policy:

* We increment the **major version** with any incompatible change to functionality, including changes to the exported Go API surface or behavior of the API.
* We increment the **minor version** with any backwards-compatible changes to functionality.
* We increment the **patch version** with any backwards-compatible bug fixes.

## License

Copyright (c) 2025 JFrog.

Apache 2.0 licensed, see [LICENSE][LICENSE] file.

[LICENSE]: ./LICENSE

