# Terraform Provider AppTrust - Project Summary

## Overview

Terraform Provider for JFrog AppTrust, providing resources and data sources to manage applications. AppTrust is part of the JFrog Platform that provides application security and compliance management capabilities as a definitive, centralized system of record for software assets throughout their lifecycle.

## Features Implemented

### Resources

1. **apptrust_application**
   - Manages AppTrust applications (create, update, delete)
   - Supports application_key, application_name, project_key, description, maturity_level, criticality, labels, user_owners, group_owners

### Data Sources

2. **apptrust_application** - Reads a single application by key.
3. **apptrust_applications** - Reads multiple applications with optional filters, pagination, and sorting.

### Provider Configuration

The provider supports:
- Artifactory URL configuration
- Access Token authentication (recommended)
- API Key authentication (deprecated, for backward compatibility)
- Environment variable support for configuration
- Version compatibility checks for Artifactory and Xray

## Project Structure

```
terraform-provider-apptrust/
├── main.go                           # Provider entry point
├── go.mod                            # Go module definition
├── go.sum                            # Go module checksums
├── GNUmakefile                       # Build and test automation
├── LICENSE                           # Apache 2.0 license
├── NOTICE                            # Third-party attributions
├── README.md                         # User documentation
├── CHANGELOG.md                      # Version history
├── CODEOWNERS                        # Code ownership
├── CONTRIBUTING.md                   # Contribution guidelines (CLA, PR process)
├── CONTRIBUTIONS.md                  # Contribution guide (building, testing)
├── PROJECT_SUMMARY.md                # This file
├── RELEASE_PROCESS.md                # Release process documentation
├── releaseAppTrustProvider.sh        # Release automation script
├── sample.tf                         # Sample Terraform configuration
├── terraform-registry-manifest.json  # Terraform registry metadata
├── pkg/apptrust/
│   ├── apptrust.go                   # Package-level utilities
│   ├── provider/
│   │   ├── framework.go              # Provider framework implementation
│   │   └── provider.go               # Provider version and constants
│   ├── resource/                     # Resource implementations
│   │   ├── resource_application.go
│   │   └── *_test.go
│   ├── datasource/                   # Data source implementations
│   │   ├── data_source_application.go
│   │   ├── data_source_applications.go
│   │   └── *_test.go
│   └── acctest/
│       └── test.go                   # Acceptance test helpers
├── docs/
│   ├── index.md                      # Provider documentation
│   ├── data-sources/                 # Data source documentation
│   └── resources/                    # Resource documentation
├── templates/
│   ├── index.md.tmpl                 # Provider doc template
│   ├── data-sources/                 # Data source doc templates
│   └── resources/                    # Resource doc templates
├── examples/
│   ├── provider/
│   ├── resources/
│   └── datasources/
└── tools/
    └── tools.go                      # Build tools
```

## Provider Configuration

The provider supports multiple authentication methods:

1. **Access Token** (Recommended) - Via configuration or `JFROG_ACCESS_TOKEN` / `ARTIFACTORY_ACCESS_TOKEN` environment variable
2. **API Key** (Deprecated) - For backward compatibility

Example configuration:

```terraform
terraform {
  required_providers {
    apptrust = {
      source  = "jfrog/apptrust"
      version = "~> 1.0"
    }
  }
}

provider "apptrust" {
  url          = "https://myinstance.jfrog.io/artifactory"
  access_token = var.jfrog_access_token
}
```

## Building the Provider

```bash
# Initialize dependencies
go mod tidy

# Build the provider
make build

# Install locally for testing
make install

# Run tests
make test

# Run acceptance tests
make acceptance

# Generate documentation
make doc
```

## Key Dependencies

| Dependency | Purpose |
|------------|---------|
| terraform-plugin-framework | Terraform provider framework |
| terraform-plugin-framework-validators | Schema validators |
| terraform-plugin-testing | Acceptance testing |
| terraform-provider-shared | JFrog shared utilities |
| go-resty/resty | HTTP client |
| samber/lo | Go utilities |

## OpenTofu Support

This provider is compatible with OpenTofu. Releases are published to both:
- Terraform Registry: `registry.terraform.io/jfrog/apptrust`
- OpenTofu Registry: `registry.opentofu.org/jfrog/apptrust`

## Development Notes

- Built with Terraform Plugin Framework
- Uses JFrog shared library for common functionality
- Compatible with Go 1.24+
- Supports Terraform 1.0+ and OpenTofu 1.0+
- All source files include Apache 2.0 copyright headers

## Current Version

See [CHANGELOG.md](./CHANGELOG.md) for version history.

## License

Apache 2.0 - Copyright (c) 2025 JFrog Ltd
