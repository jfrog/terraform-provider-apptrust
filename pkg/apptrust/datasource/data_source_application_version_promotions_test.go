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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

// TestAccApplicationVersionPromotionsDataSource_basic creates app, version, optional promotion, then reads promotions.
// Set APPTRUST_TEST_TARGET_STAGE to run with promotion (e.g. QA); otherwise only app+version+datasource are created.
func TestAccApplicationVersionPromotionsDataSource_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	targetStage := os.Getenv("APPTRUST_TEST_TARGET_STAGE")
	withPromotion := targetStage != ""

	id, appFqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	versionId, versionFqrn, versionName := testutil.MkNames("test-ver-", "apptrust_application_version")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	version := fmt.Sprintf("1.0.%d", versionId)
	dataSourceFqrn := "data.apptrust_application_version_promotions.test"

	var config string
	if withPromotion {
		_, _, promoName := testutil.MkNames("test-promo-", "apptrust_application_version_promotion")
		config = fmt.Sprintf(`
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
			resource "apptrust_application_version_promotion" "%s" {
				application_key = apptrust_application_version.%s.application_key
				version        = apptrust_application_version.%s.version
				target_stage   = "%s"
				promotion_type = "copy"
			}
			data "apptrust_application_version_promotions" "test" {
				application_key = apptrust_application_version.%s.application_key
				version         = apptrust_application_version.%s.version
			}
		`, appName, appKey, appName, projectKey, versionName, appName, version, promoName, versionName, versionName, targetStage, versionName, versionName)
	} else {
		config = fmt.Sprintf(`
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
			data "apptrust_application_version_promotions" "test" {
				application_key = apptrust_application_version.%s.application_key
				version         = apptrust_application_version.%s.version
			}
		`, appName, appKey, appName, projectKey, versionName, appName, version, versionName, versionName)
	}

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(dataSourceFqrn, "application_key", appKey),
		resource.TestCheckResourceAttr(dataSourceFqrn, "version", version),
		resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
		resource.TestCheckResourceAttrSet(dataSourceFqrn, "promotions.#"),
	}
	if withPromotion {
		checks = append(checks, resource.TestCheckResourceAttr(dataSourceFqrn, "total", "1"))
	}

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
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
