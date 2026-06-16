## 1.1.0 (June 16, 2026)

FEATURES:

* **New Data Source:** `apptrust_projects_stages` PR: [#26](https://github.com/jfrog/terraform-provider-apptrust/pull/26)

## 1.0.2 (Apr 16, 2026)

BUG FIXES:

* `apptrust_application` — Fixed labels serialization: labels were incorrectly sent as a JSON map (`{"key":"value"}`) instead of the API-required array format (`[{"key":"k","value":"v"}]`). This caused label create/update operations to fail silently or return 400 errors.
* `apptrust_application` — Removed erroneous `project` query parameter that was being appended to PATCH requests.
* `apptrust_application` (datasource) — Fixed labels deserialization from array-of-objects format returned by the API.
* `apptrust_applications` (datasource) — Fixed per-application `labels` deserialization from array-of-objects to map.
* `apptrust_applications` (datasource) — Read the paginated API wrapper (`applications`, `total`, `limit`, `offset`) instead of unmarshaling the response as a bare array; list metadata now matches the server.
* `apptrust_bound_package_versions` (datasource) — On not-found, set `versions` to an empty list instead of null so Terraform state stays consistent.

ENHANCEMENTS:

* `apptrust_application_version` — Added missing source types: `source_release_bundles`, `source_packages`, `source_aql`, `skip_docker_manifest_resolution`. Previously only `source_artifacts`, `source_builds`, and `source_versions` were supported.
* `apptrust_application_version` — Added `draft` attribute support for creating draft versions.
* `apptrust_application_version` — Added `filter_included` and `filter_excluded` artifact filter attributes.
* `apptrust_applications` (datasource) — The `applications` list now returns full application details (`description`, `maturity_level`, `criticality`, `labels`, `user_owners`, `group_owners`) instead of placeholder version fields that were always empty.
* `apptrust_application_version_promotion` — When create runs immediately after version create, retry promotion with backoff if the API returns 400/422 while the version is still `STARTED` or `PROCESSING` (async processing).
* `acctest` — Project key constants (`AppTrustProjectKey1/2/3/4`) are now configurable via `APPTRUST_PROJECT_KEY_1/2/3/4` environment variables (default: `testproj3038182`).
* `acctest` — Added `EnsureTestArtifact` helper to upload a test artifact needed by promotion acceptance tests; promotion and rollback tests register lifecycle via `terraform-provider-platform` and target the global `DEV` stage (removed reliance on `APPTRUST_TEST_TARGET_STAGE`).

FEATURES:

* New data source `apptrust_application_version` — Reads full details of a specific application version (tag, status, release_status, current_stage, created_by, created) by application key and version identifier.

NOTES:

* Generated documentation and examples updated for the new data source, `apptrust_applications` schema changes, and `apptrust_application_version` optional attributes.

## 1.0.1 (Feb 23, 2026)

NOTES:

* Documentation-only release: expanded provider index and examples README; refreshed generated docs and templates across data sources and resources.

## 1.0.0 (Feb 23, 2026)

This release includes all resources and datasources for the AppTrust provider.

FEATURES:

**Resources:**

* `apptrust_application` — Manages the full lifecycle of an AppTrust application (create, update, delete). Attributes: `application_key`, `application_name`, `project_key`, `description`, `maturity_level` (development / staging / production / deprecated), `criticality` (low / medium / high / critical), `labels` (key-value), `user_owners`, `group_owners`. Computed: `id`.
* `apptrust_application_version` — Creates and manages an application version. Attributes: `application_key`, `version`, `tag`, `source_artifacts`, `source_builds`, `source_versions`, `properties`. Computed: `release_status`, `current_stage`.
* `apptrust_application_version_promotion` — Promotes an application version to a lifecycle stage (e.g. staging, production). Attributes: `application_key`, `version`, target stage and optional comment.
* `apptrust_application_version_release` — Releases an application version (marks as released / trusted release). Attributes: `application_key`, `version`.
* `apptrust_application_version_rollback` — Rolls back an application version to a previous stage. Attributes: `application_key`, `version`.
* `apptrust_bound_package` — Binds a package version to an AppTrust application. Attributes: `application_key`, `package_type`, `package_name`, `package_version`. A package version can be bound to only one application.

**Data Sources:**

* `apptrust_application` — Reads a single application by `project_key` and `application_key`. Returns `application_name`, `description`, `maturity_level`, `criticality`, `labels`, `user_owners`, and `group_owners`.
* `apptrust_applications` — Reads multiple applications with optional filters: `project_key`, `name`, `criticality`, `maturity`, `labels`, `owner`. Supports pagination (`limit`, `offset`) and sorting (`order_by`, `order_asc`).
* `apptrust_application_versions` — Lists versions for an application. Optional filters and pagination.
* `apptrust_application_version_status` — Reads the status of a specific application version (release status, current stage).
* `apptrust_application_version_promotions` — Lists promotions for an application version. Pagination and filters supported.
* `apptrust_application_package_bindings` — Lists package bindings for an application (bound packages).
* `apptrust_bound_package_versions` — Lists versions for a bound package (by application, package type, and name).
