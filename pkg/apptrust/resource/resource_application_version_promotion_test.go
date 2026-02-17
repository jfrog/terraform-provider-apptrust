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

// TestAccApplicationVersionPromotion_basic creates an application, version, and promotes to a target stage.
// Requires lifecycle stage (e.g. QA) to exist in the project. Set APPTRUST_TEST_TARGET_STAGE to match your project.
func TestAccApplicationVersionPromotion_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	targetStage := os.Getenv("APPTRUST_TEST_TARGET_STAGE")
	if targetStage == "" {
		targetStage = "QA"
	}

	id, appFqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	versionId, versionFqrn, versionName := testutil.MkNames("test-ver-", "apptrust_application_version")
	_, promoFqrn, promoName := testutil.MkNames("test-promo-", "apptrust_application_version_promotion")
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
			application_key   = apptrust_application.%s.application_key
			version           = "%s"
			tag               = "acc-test"
			source_artifacts  = [{ path = "generic-repo/readme.md" }]
		}
		resource "apptrust_application_version_promotion" "%s" {
			application_key = apptrust_application_version.%s.application_key
			version        = apptrust_application_version.%s.version
			target_stage   = "%s"
			promotion_type = "copy"
		}
	`, appName, appKey, appName, projectKey, versionName, appName, version, promoName, versionName, versionName, targetStage)

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
					resource.TestCheckResourceAttr(promoFqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(promoFqrn, "version", version),
					resource.TestCheckResourceAttr(promoFqrn, "target_stage", targetStage),
					resource.TestCheckResourceAttr(promoFqrn, "promotion_type", "copy"),
					resource.TestCheckResourceAttrSet(promoFqrn, "id"),
				),
			},
			{
				ResourceName:      promoFqrn,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s:%s:%s", appKey, version, targetStage),
			},
		},
	})
}
