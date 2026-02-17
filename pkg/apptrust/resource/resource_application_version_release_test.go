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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

// TestAccApplicationVersionRelease_basic creates an application, version, and releases to PROD.
// Requires PROD stage and release policies in the project. Set APPTRUST_TEST_RELEASE=1 to run.
func TestAccApplicationVersionRelease_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)
	if os.Getenv("APPTRUST_TEST_RELEASE") == "" {
		t.Skip("Set APPTRUST_TEST_RELEASE=1 to run application version release acceptance test (requires PROD stage)")
	}

	id, appFqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	versionId, versionFqrn, versionName := testutil.MkNames("test-ver-", "apptrust_application_version")
	_, releaseFqrn, releaseName := testutil.MkNames("test-release-", "apptrust_application_version_release")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	version := fmt.Sprintf("1.0.%d", versionId)

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
		resource "apptrust_application_version" "%s" {
			application_key  = apptrust_application.%s.application_key
			version          = "%s"
			tag              = "acc-release"
			source_artifacts = [{ path = "generic-repo/readme.md" }]
		}
		resource "apptrust_application_version_release" "%s" {
			application_key = apptrust_application_version.%s.application_key
			version         = apptrust_application_version.%s.version
			promotion_type  = "copy"
		}
	`, appName, appKey, appName, projectKey, versionName, appName, version, releaseName, versionName, versionName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckApplicationVersionDestroy(versionFqrn),
			testAccCheckApplicationDestroy(appFqrn),
		),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(releaseFqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(releaseFqrn, "version", version),
					resource.TestCheckResourceAttrSet(releaseFqrn, "id"),
				),
			},
			{
				ResourceName:      releaseFqrn,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s:%s", appKey, version),
			},
		},
	})
}
