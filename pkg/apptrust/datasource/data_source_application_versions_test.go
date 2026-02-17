// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datasource_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

func TestAccApplicationVersionsDataSource_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	dataSourceFqrn := "data.apptrust_application_versions.test"

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
		data "apptrust_application_versions" "test" {
			application_key = apptrust_application.%s.application_key
		}
	`, appName, appKey, appName, projectKey, appName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceFqrn, "application_key", appKey),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "versions.#"),
				),
			},
		},
	})
}

func TestAccApplicationVersionsDataSource_pagination(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	page1Fqrn := "data.apptrust_application_versions.page1"
	page2Fqrn := "data.apptrust_application_versions.page2"
	const pageSize = 5

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
		data "apptrust_application_versions" "page1" {
			application_key = apptrust_application.%s.application_key
			limit           = %d
			offset          = 0
			order_asc       = false
		}
		data "apptrust_application_versions" "page2" {
			application_key = apptrust_application.%s.application_key
			limit           = %d
			offset          = %d
			order_asc       = false
		}
	`, appName, appKey, appName, projectKey, appName, pageSize, appName, pageSize, pageSize)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(page1Fqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(page2Fqrn, "application_key", appKey),
					resource.TestCheckResourceAttrSet(page1Fqrn, "total"),
					resource.TestCheckResourceAttrSet(page1Fqrn, "versions.#"),
					resource.TestCheckResourceAttrSet(page2Fqrn, "total"),
					resource.TestCheckResourceAttrSet(page2Fqrn, "versions.#"),
					testAccCheckApplicationVersionsPaginationTotalMatches(page1Fqrn, page2Fqrn),
					testAccCheckApplicationVersionsPageSize(page1Fqrn, pageSize),
					testAccCheckApplicationVersionsPageSize(page2Fqrn, pageSize),
				),
			},
		},
	})
}

// testAccCheckApplicationVersionsPaginationTotalMatches verifies two application_versions datasources have the same total.
func testAccCheckApplicationVersionsPaginationTotalMatches(fqrn1, fqrn2 string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs1, ok := s.RootModule().Resources[fqrn1]
		if !ok {
			return fmt.Errorf("datasource not found: %s", fqrn1)
		}
		rs2, ok := s.RootModule().Resources[fqrn2]
		if !ok {
			return fmt.Errorf("datasource not found: %s", fqrn2)
		}
		t1 := rs1.Primary.Attributes["total"]
		t2 := rs2.Primary.Attributes["total"]
		if t1 != t2 {
			return fmt.Errorf("total mismatch: %s has total=%s, %s has total=%s", fqrn1, t1, fqrn2, t2)
		}
		return nil
	}
}

// testAccCheckApplicationVersionsPageSize verifies versions count does not exceed the requested limit.
func testAccCheckApplicationVersionsPageSize(fqrn string, maxCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[fqrn]
		if !ok {
			return fmt.Errorf("datasource not found: %s", fqrn)
		}
		countStr, ok := rs.Primary.Attributes["versions.#"]
		if !ok {
			return fmt.Errorf("versions.# not found for %s", fqrn)
		}
		var count int
		if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil {
			return fmt.Errorf("versions.# invalid: %q", countStr)
		}
		if count > maxCount {
			return fmt.Errorf("versions.# = %d exceeds limit %d", count, maxCount)
		}
		return nil
	}
}
