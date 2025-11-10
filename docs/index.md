# AppTrust Provider

The [AppTrust](https://jfrog.com/apptrust/) provider is used to interact with the AppTrust resources supported by JFrog Artifactory. The provider needs to be configured with the proper credentials before it can be used.

Links to documentation for specific resources can be found in the table of contents to the left.

This provider requires access to AppTrust APIs, which are only available with:

- **Enterprise Plus (E+)** license tier
- **AppTrust entitlements** enabled on the license

AppTrust is an add-on module that requires specific entitlements. Not all Enterprise Plus licenses automatically include AppTrust - the license must explicitly include AppTrust entitlements.

You can determine which license you have by accessing the following URL `${host}/artifactory/api/system/licenses/`.

You can either access it via API, or web browser - it requires admin level credentials.

```sh
curl -sL ${host}/artifactory/api/system/licenses/ | jq .
{
  "type" : "Enterprise Plus Trial",
  "validThrough" : "Jan 29, 2022",
  "licensedTo" : "JFrog Ltd"
}
```

**Note:** If AppTrust is not licensed or entitlements are missing, AppTrust API calls will return appropriate errors. The provider does not validate license availability during initialization - errors will be surfaced when API operations are attempted.

## Terraform CLI version support

Current version supports [Terraform Protocol v6](https://developer.hashicorp.com/terraform/plugin/terraform-plugin-protocol#protocol-version-6) which means Terraform CLI version 1.0 and later.

## Requirements

- Artifactory 7.125.x or later
- Xray 3.130.5 or later
- AppTrust license

## Example Usage

```tf
# Required for Terraform 1.0 and up (https://www.terraform.io/upgrade-guides)
terraform {
  required_providers {
    apptrust = {
      source  = "jfrog/apptrust"
      version = "~> 1.0"
    }
  }
}

# Configure the AppTrust provider
provider "apptrust" {
  url           = "${var.artifactory_url}/artifactory"
  access_token  = "${var.artifactory_access_token}"
}

# Create a new application
resource "apptrust_application" "my_app" {
  application_name = "My Application"
  application_key  = "my-app"
  project_key     = "my-project"
  description     = "A sample application"
  maturity_level  = "production"
  criticality     = "high"
}
```

## Authentication

The AppTrust provider supports the following authentication methods:

### Access Token

AppTrust access tokens may be used via the Authorization header by providing the `access_token` attribute to the provider block. Getting this value from the environment is supported with `JFROG_ACCESS_TOKEN` or `ARTIFACTORY_ACCESS_TOKEN` environment variables.

```tf
provider "apptrust" {
  url          = "https://your-instance.jfrog.io/artifactory"
  access_token = "your-access-token"
}
```

### API Key (Deprecated)

API keys are deprecated but still supported for backward compatibility. Getting this value from the environment is supported with `ARTIFACTORY_API_KEY` or `JFROG_API_KEY` environment variables.

## Configuration Reference

The following arguments are supported:

- `url` - (Optional) The base URL for the Artifactory instance. Can also be set via the `ARTIFACTORY_URL` or `JFROG_URL` environment variables.
- `access_token` - (Optional) The Artifactory access token. Can also be set via the `ARTIFACTORY_ACCESS_TOKEN` or `JFROG_ACCESS_TOKEN` environment variables.
- `api_key` - (Optional, Deprecated) The Artifactory API key. Can also be set via the `ARTIFACTORY_API_KEY` or `JFROG_API_KEY` environment variables.
