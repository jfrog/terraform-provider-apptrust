## 1.0.0 (Feb 23, 2025). Tested on Artifactory 7.125.0 with Terraform 1.0+ and OpenTofu 1.0+

This release includes all resources and datasources for the AppTrust provider.

FEATURES:

**Resources:**

* `apptrust_application` — Manages the full lifecycle of an AppTrust application (create, update, delete). Attributes: `project_key`, `name`, `description`, `owner`, `criticality` (low / medium / high / critical), `maturity` (development / staging / production / deprecated), and `labels` (key-value).
* `apptrust_application_version` — Creates and manages an application version. Attributes: `application_key`, `version`, `tag`, `source_artifacts`, `source_builds`, `source_versions`, `properties`. Computed: `release_status`, `current_stage`.
* `apptrust_application_version_promotion` — Promotes an application version to a lifecycle stage (e.g. staging, production). Attributes: `application_key`, `version`, target stage and optional comment.
* `apptrust_application_version_release` — Releases an application version (marks as released / trusted release). Attributes: `application_key`, `version`.
* `apptrust_application_version_rollback` — Rolls back an application version to a previous stage. Attributes: `application_key`, `version`.
* `apptrust_bound_package` — Binds a package version to an AppTrust application. Attributes: `application_key`, `package_type`, `package_name`, `package_version`. A package version can be bound to only one application.

**Data Sources:**

* `apptrust_application` — Reads a single application by `project_key` and application `key`. Returns application attributes (name, description, owner, criticality, maturity, labels).
* `apptrust_applications` — Reads multiple applications with optional filters: `project_key`, `name`, `criticality`, `maturity`, `labels`, `owner`. Supports pagination (`limit`, `offset`) and sorting (`order_by`, `order_asc`).
* `apptrust_application_versions` — Lists versions for an application. Optional filters and pagination.
* `apptrust_application_version_status` — Reads the status of a specific application version (release status, current stage).
* `apptrust_application_version_promotions` — Lists promotions for an application version. Pagination and filters supported.
* `apptrust_application_package_bindings` — Lists package bindings for an application (bound packages).
* `apptrust_bound_package_versions` — Lists versions for a bound package (by application, package type, and name).
