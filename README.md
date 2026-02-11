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

Detailed documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/jfrog/apptrust/latest/docs). Summary:

### Resources

| Resource | Description |
|----------|-------------|
| **apptrust_application** | Manages AppTrust applications. Create, update, and delete applications with labels, owners, maturity level, and criticality. |

### Data Sources

| Data Source | Description |
|-------------|-------------|
| **apptrust_application** | Reads a single application by key. |
| **apptrust_applications** | Reads multiple applications with optional filters, pagination, and sorting. |

## Local Development

For local development, you can use `dev_overrides` to test the provider without publishing it to the registry.

### Quick Setup

1. **Set up dev_overrides** (one-time setup):
   ```bash
   ./setup-dev-overrides.sh
   ```
   Or manually create/update `~/.terraformrc`:
   ```hcl
   provider_installation {
     dev_overrides {
       "jfrog/apptrust" = "/absolute/path/to/terraform-provider-apptrust"
     }
     direct {}
   }
   ```

2. **Build and install the provider**:
   ```bash
   make install
   ```

3. **Use Terraform commands directly** (no need for `terraform init`):
   ```bash
   terraform validate
   terraform plan
   terraform apply
   ```

See [CONTRIBUTIONS.md](CONTRIBUTIONS.md) for contribution guidelines and [CONTRIBUTING.md](CONTRIBUTING.md) for CLA and pull request process.

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

This provider uses the JFrog Artifactory REST API to interact with AppTrust features (`/apptrust/api/v1/applications`) to manage applications as a system of record for software assets throughout their lifecycle.

## Contributors

See the [contribution guide](CONTRIBUTIONS.md).

## Versioning

In general, this project follows [semver](https://semver.org/) as closely as we can for tagging releases of the package. We've adopted the following versioning policy:

* We increment the **major version** with any incompatible change to functionality, including changes to the exported Go API surface or behavior of the API.
* We increment the **minor version** with any backwards-compatible changes to functionality.
* We increment the **patch version** with any backwards-compatible bug fixes.

## License

Copyright (c) 2025 JFrog.

Apache 2.0 licensed, see [LICENSE][LICENSE] file.

[LICENSE]: ./LICENSE
