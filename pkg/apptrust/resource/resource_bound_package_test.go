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

package resource_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

const applicationPackagesEndpoint = "apptrust/api/v1/applications"

func TestAccBoundPackage_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	pkgType := os.Getenv("APPTRUST_TEST_PACKAGE_TYPE")
	pkgName := os.Getenv("APPTRUST_TEST_PACKAGE_NAME")
	pkgVersion := os.Getenv("APPTRUST_TEST_PACKAGE_VERSION")
	if pkgType == "" || pkgName == "" || pkgVersion == "" {
		t.Skip("Set APPTRUST_TEST_PACKAGE_TYPE, APPTRUST_TEST_PACKAGE_NAME, APPTRUST_TEST_PACKAGE_VERSION to run bound package acceptance test")
	}

	id, appFqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	_, pkgFqrn, pkgNameRes := testutil.MkNames("test-pkg-", "apptrust_bound_package")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
		resource "apptrust_bound_package" "%s" {
			application_key   = apptrust_application.%s.application_key
			package_type     = "%s"
			package_name     = "%s"
			package_version  = "%s"
		}
	`, appName, appKey, appName, projectKey, pkgNameRes, appName, pkgType, pkgName, pkgVersion)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckBoundPackageDestroy(pkgFqrn),
			testAccCheckApplicationDestroy(appFqrn),
		),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(pkgFqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(pkgFqrn, "package_type", pkgType),
					resource.TestCheckResourceAttr(pkgFqrn, "package_name", pkgName),
					resource.TestCheckResourceAttr(pkgFqrn, "package_version", pkgVersion),
					resource.TestCheckResourceAttrSet(pkgFqrn, "id"),
				),
			},
			{
				ResourceName:      pkgFqrn,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s:%s:%s:%s", appKey, pkgType, pkgName, pkgVersion),
			},
		},
	})
}

func testAccCheckBoundPackageDestroy(fqrn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[fqrn]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return nil
		}
		appKey := rs.Primary.Attributes["application_key"]
		pkgType := rs.Primary.Attributes["package_type"]
		pkgName := rs.Primary.Attributes["package_name"]
		version := rs.Primary.Attributes["package_version"]
		if appKey == "" || pkgType == "" || pkgName == "" || version == "" {
			return nil
		}
		client, err := acctest.GetTestRestyFromEnv()
		if err != nil {
			return err
		}
		var listResp struct {
			Versions []struct {
				Version string `json:"version"`
			} `json:"versions"`
		}
		resp, err := client.R().
			SetPathParam("application_key", appKey).
			SetPathParam("type", pkgType).
			SetPathParam("name", pkgName).
			SetResult(&listResp).
			Get(applicationPackagesEndpoint + "/{application_key}/packages/{type}/{name}")
		if err != nil {
			return err
		}
		if resp.StatusCode() == http.StatusNotFound {
			return nil
		}
		if !resp.IsSuccess() {
			return nil
		}
		for _, v := range listResp.Versions {
			if v.Version == version {
				return fmt.Errorf("bound package %s %s:%s still exists", appKey, pkgName, version)
			}
		}
		return nil
	}
}
