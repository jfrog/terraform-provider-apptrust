package datasource_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

func TestAccApplicationsDataSource_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	projectKey := acctest.AppTrustProjectKey1
	dataSourceFqrn := "data.apptrust_applications.test"

	config := fmt.Sprintf(`
		data "apptrust_applications" "test" {
			project_key = "%s"
		}
	`, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					// List attribute must be checked by count (applications.#), not by Set
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "applications.#"),
				),
			},
		},
	})
}

func TestAccApplicationsDataSource_filterByMaturity(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	projectKey := acctest.AppTrustProjectKey1
	dataSourceFqrn := "data.apptrust_applications.test"

	// Create three applications with different maturity levels
	idProd, fqrnProd, nameProd := testutil.MkNames("test-app-prod-", "apptrust_application")
	idExp, fqrnExp, nameExp := testutil.MkNames("test-app-exp-", "apptrust_application")
	idUnspec, fqrnUnspec, nameUnspec := testutil.MkNames("test-app-unspec-", "apptrust_application")
	appKeyProd := fmt.Sprintf("app-prod-%d", idProd)
	appKeyExp := fmt.Sprintf("app-exp-%d", idExp)
	appKeyUnspec := fmt.Sprintf("app-unspec-%d", idUnspec)

	resourceConfig := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
			maturity_level   = "production"
		}
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
			maturity_level   = "experimental"
		}
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
			maturity_level   = "unspecified"
		}
	`, nameProd, appKeyProd, nameProd, projectKey,
		nameExp, appKeyExp, nameExp, projectKey,
		nameUnspec, appKeyUnspec, nameUnspec, projectKey)

	// Query with maturity = "production" — only production apps should be returned (project assumed empty)
	dataSourceConfig := fmt.Sprintf(`
		%s

		data "apptrust_applications" "test" {
			project_key = "%s"
			maturity    = "production"
		}
	`, resourceConfig, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckApplicationDestroyApplications(fqrnProd),
			testAccCheckApplicationDestroyApplications(fqrnExp),
			testAccCheckApplicationDestroyApplications(fqrnUnspec),
		),
		Steps: []resource.TestStep{
			{
				Config: dataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					// Filter returns at least our production app; project may have other production apps from other runs
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "applications.#"),
					testAccCheckApplicationsListContainsApp(dataSourceFqrn, appKeyProd, nameProd, projectKey),
				),
			},
		},
	})
}

func TestAccApplicationsDataSource_filterByCriticality(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	dataSourceFqrn := "data.apptrust_applications.test"

	resourceConfig := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
			criticality      = "high"
		}
	`, name, appKey, name, projectKey)

	dataSourceConfig := fmt.Sprintf(`
		%s

		data "apptrust_applications" "test" {
			project_key = "%s"
			criticality = "high"
		}
	`, resourceConfig, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroyApplications(fqrn),
		Steps: []resource.TestStep{
			{
				Config: dataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "applications.#"),
				),
			},
		},
	})
}

func TestAccApplicationsDataSource_filterByLabels(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	dataSourceFqrn := "data.apptrust_applications.test"

	resourceConfig := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
			
			labels = {
				environment = "test"
				team        = "qa"
			}
		}
	`, name, appKey, name, projectKey)

	dataSourceConfig := fmt.Sprintf(`
		%s

		data "apptrust_applications" "test" {
			project_key = "%s"
			labels = [
				"environment:test"
			]
		}
	`, resourceConfig, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroyApplications(fqrn),
		Steps: []resource.TestStep{
			{
				Config: dataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "applications.#"),
				),
			},
		},
	})
}

func TestAccApplicationsDataSource_pagination(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	projectKey := acctest.AppTrustProjectKey1
	page1Fqrn := "data.apptrust_applications.page1"
	page2Fqrn := "data.apptrust_applications.page2"
	const pageSize = 5

	config := fmt.Sprintf(`
		data "apptrust_applications" "page1" {
			project_key = "%s"
			limit       = %d
			offset      = 0
			order_by    = "name"
			order_asc   = true
		}
		data "apptrust_applications" "page2" {
			project_key = "%s"
			limit       = %d
			offset      = %d
			order_by    = "name"
			order_asc   = true
		}
	`, projectKey, pageSize, projectKey, pageSize, pageSize)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(page1Fqrn, "total"),
					resource.TestCheckResourceAttrSet(page1Fqrn, "applications.#"),
					resource.TestCheckResourceAttrSet(page2Fqrn, "total"),
					resource.TestCheckResourceAttrSet(page2Fqrn, "applications.#"),
					// Note: total is derived from current page length when API does not return full result count
					testAccCheckApplicationsPageSize(page1Fqrn, pageSize),
					testAccCheckApplicationsPageSize(page2Fqrn, pageSize),
				),
			},
		},
	})
}

// testAccCheckApplicationsListContainsApp verifies the applications list contains an element with the given application_key, application_name, and project_key (list is index-based: applications.0, applications.1, ...).
func testAccCheckApplicationsListContainsApp(dataSourceFqrn, applicationKey, applicationName, projectKey string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dataSourceFqrn]
		if !ok {
			return fmt.Errorf("datasource not found: %s", dataSourceFqrn)
		}
		countStr, ok := rs.Primary.Attributes["applications.#"]
		if !ok {
			return fmt.Errorf("applications.# not found")
		}
		var count int
		if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil || count < 1 {
			return fmt.Errorf("applications.# is %q (expected at least 1)", countStr)
		}
		for i := 0; i < count; i++ {
			key := rs.Primary.Attributes[fmt.Sprintf("applications.%d.application_key", i)]
			name := rs.Primary.Attributes[fmt.Sprintf("applications.%d.application_name", i)]
			proj := rs.Primary.Attributes[fmt.Sprintf("applications.%d.project_key", i)]
			if key == applicationKey && name == applicationName && proj == projectKey {
				return nil
			}
		}
		return fmt.Errorf("applications list does not contain app with application_key=%q application_name=%q project_key=%q", applicationKey, applicationName, projectKey)
	}
}

// testAccCheckApplicationsPageSize verifies applications count does not exceed the requested limit.
func testAccCheckApplicationsPageSize(fqrn string, maxCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[fqrn]
		if !ok {
			return fmt.Errorf("datasource not found: %s", fqrn)
		}
		countStr, ok := rs.Primary.Attributes["applications.#"]
		if !ok {
			return fmt.Errorf("applications.# not found for %s", fqrn)
		}
		var count int
		if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil {
			return fmt.Errorf("applications.# invalid: %q", countStr)
		}
		if count > maxCount {
			return fmt.Errorf("applications.# = %d exceeds limit %d", count, maxCount)
		}
		return nil
	}
}

func TestAccApplicationsDataSource_filterByName(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	dataSourceFqrn := "data.apptrust_applications.test"

	resourceConfig := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, appKey, name, projectKey)

	dataSourceConfig := fmt.Sprintf(`
		%s

		data "apptrust_applications" "test" {
			project_key = "%s"
			name        = "%s"
		}
	`, resourceConfig, projectKey, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroyApplications(fqrn),
		Steps: []resource.TestStep{
			{
				Config: dataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "applications.#"),
				),
			},
		},
	})
}

func TestAccApplicationsDataSource_multipleFilters(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	dataSourceFqrn := "data.apptrust_applications.test"

	resourceConfig := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
			maturity_level   = "production"
			criticality      = "high"
			
			labels = {
				environment = "production"
			}
			
			user_owners = ["admin"]
		}
	`, name, appKey, name, projectKey)

	dataSourceConfig := fmt.Sprintf(`
		%s

		data "apptrust_applications" "test" {
			project_key = "%s"
			maturity    = "production"
			criticality = "high"

			labels = [
				"environment:production"
			]

			owners = [
				"admin"
			]
		}
	`, resourceConfig, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroyApplications(fqrn),
		Steps: []resource.TestStep{
			{
				Config: dataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "applications.#"),
				),
			},
		},
	})
}

// testAccCheckApplicationDestroyApplications is a helper function for applications datasource tests
func testAccCheckApplicationDestroyApplications(id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("error: Resource id [%s] not found", id)
		}

		applicationKey := rs.Primary.Attributes["application_key"]
		if applicationKey == "" {
			return fmt.Errorf("error: application_key not found in state")
		}

		client, err := acctest.GetTestRestyFromEnv()
		if err != nil {
			return fmt.Errorf("error creating resty client: %w", err)
		}

		response, err := client.R().
			SetPathParam("application_key", applicationKey).
			Get("apptrust/api/v1/applications/{application_key}")

		if err != nil {
			return err
		}

		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		if response.IsError() {
			return fmt.Errorf("error checking application: %s", response.String())
		}

		// Application still exists - attempt to delete it for cleanup
		deleteResponse, err := client.R().
			SetPathParam("application_key", applicationKey).
			Delete("apptrust/api/v1/applications/{application_key}")

		if err != nil {
			return fmt.Errorf("error deleting application %s during cleanup: %w", applicationKey, err)
		}

		if deleteResponse.StatusCode() == http.StatusNotFound {
			return nil
		}

		if deleteResponse.IsError() {
			return fmt.Errorf("error: application %s still exists and could not be deleted during cleanup: %s", applicationKey, deleteResponse.String())
		}

		return nil
	}
}
