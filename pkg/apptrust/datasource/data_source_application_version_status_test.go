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
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

const applicationVersionsEndpoint = "apptrust/api/v1/applications"

func TestAccApplicationVersionStatusDataSource_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, appFqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	versionId, versionFqrn, versionName := testutil.MkNames("test-ver-", "apptrust_application_version")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	version := fmt.Sprintf("1.0.%d", versionId)
	dataSourceFqrn := "data.apptrust_application_version_status.test"

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
		resource "apptrust_application_version" "%s" {
			application_key  = apptrust_application.%s.application_key
			version          = "%s"
			tag              = "acc-test"
			source_artifacts = [{ path = "generic-repo/readme.md" }]
		}
		data "apptrust_application_version_status" "test" {
			application_key = apptrust_application_version.%s.application_key
			version         = apptrust_application_version.%s.version
		}
	`, appName, appKey, appName, projectKey, versionName, appName, version, versionName, versionName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckApplicationVersionDestroyDatasource(versionFqrn),
			testAccCheckApplicationDestroy(appFqrn),
		),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceFqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(dataSourceFqrn, "version", version),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "version_release_status"),
				),
			},
		},
	})
}

func testAccCheckApplicationVersionDestroyDatasource(fqrn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[fqrn]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}
		appKey := rs.Primary.Attributes["application_key"]
		version := rs.Primary.Attributes["version"]
		if appKey == "" || version == "" {
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
			SetResult(&listResp).
			Get(applicationVersionsEndpoint + "/{application_key}/versions")
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
				return fmt.Errorf("application version %s:%s still exists", appKey, version)
			}
		}
		return nil
	}
}
