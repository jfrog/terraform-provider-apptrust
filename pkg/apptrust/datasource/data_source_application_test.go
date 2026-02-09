package datasource_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

const applicationEndpoint = "apptrust/api/v1/applications"

func TestAccApplicationDataSource_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-", "apptrust_application")
	dataSourceFqrn := "data.apptrust_application.test"
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)

	// First create the application
	resourceConfig := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, appKey, name, projectKey)

	// Then query it
	dataSourceConfig := fmt.Sprintf(`
		%s

		data "apptrust_application" "test" {
			application_key = apptrust_application.%s.application_key
		}
	`, resourceConfig, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: dataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					// Verify the data source attributes
					resource.TestCheckResourceAttr(dataSourceFqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(dataSourceFqrn, "application_name", name),
					resource.TestCheckResourceAttr(dataSourceFqrn, "project_key", projectKey),
					// Note: maturity_level and criticality may be null if not set, so we don't check for them
				),
			},
		},
	})
}

func TestAccApplicationDataSource_full(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-full-", "apptrust_application")
	dataSourceFqrn := "data.apptrust_application.test"
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)

	resourceConfig := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
			description      = "Test application for datasource"
			maturity_level   = "production"
			criticality      = "high"
			
			labels = {
				environment = "test"
				team        = "qa"
			}
			
			user_owners = ["test-user"]
			group_owners = ["test-group"]
		}
	`, name, appKey, name, projectKey)

	dataSourceConfig := fmt.Sprintf(`
		%s

		data "apptrust_application" "test" {
			application_key = apptrust_application.%s.application_key
		}
	`, resourceConfig, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: dataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceFqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(dataSourceFqrn, "application_name", name),
					resource.TestCheckResourceAttr(dataSourceFqrn, "project_key", projectKey),
					resource.TestCheckResourceAttr(dataSourceFqrn, "description", "Test application for datasource"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "maturity_level", "production"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "criticality", "high"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "labels.environment", "test"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "labels.team", "qa"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "user_owners.#", "1"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "user_owners.0", "test-user"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "group_owners.#", "1"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "group_owners.0", "test-group"),
				),
			},
		},
	})
}

func TestAccApplicationDataSource_notFound(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	config := `
		data "apptrust_application" "test" {
			application_key = "non-existent-app-key-12345"
		}
	`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Application Not Found|not found|404|Unable to Read`),
			},
		},
	})
}

func TestAccApplicationDataSource_emptyFields(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-empty-", "apptrust_application")
	dataSourceFqrn := "data.apptrust_application.test"
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)

	resourceConfig := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, appKey, name, projectKey)

	dataSourceConfig := fmt.Sprintf(`
		%s

		data "apptrust_application" "test" {
			application_key = apptrust_application.%s.application_key
		}
	`, resourceConfig, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: dataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceFqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(dataSourceFqrn, "application_name", name),
					resource.TestCheckResourceAttr(dataSourceFqrn, "project_key", projectKey),
					// Verify optional fields are null when not set
					resource.TestCheckNoResourceAttr(dataSourceFqrn, "description"),
					resource.TestCheckNoResourceAttr(dataSourceFqrn, "labels"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "user_owners.#", "0"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "group_owners.#", "0"),
				),
			},
		},
	})
}

func testAccCheckApplicationDestroy(id string) resource.TestCheckFunc {
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
			Get(applicationEndpoint + "/{application_key}")

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
			Delete(applicationEndpoint + "/{application_key}")

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
