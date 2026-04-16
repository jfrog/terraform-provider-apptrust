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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

// TestAccApplicationVersionRollback_basic provisions a lifecycle via terraform-provider-platform
// then creates app -> version -> promotion to DEV -> rollback from DEV.
func TestAccApplicationVersionRollback_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)
	acctest.EnsureTestArtifact(t)

	id, appFqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	versionId, versionFqrn, versionName := testutil.MkNames("test-ver-", "apptrust_application_version")
	_, _, promoName := testutil.MkNames("test-promo-", "apptrust_application_version_promotion")
	_, rollbackFqrn, rollbackName := testutil.MkNames("test-rollback-", "apptrust_application_version_rollback")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	version := fmt.Sprintf("1.0.%d", versionId)
	const targetStage = "DEV"

	jfrogURL := acctest.GetArtifactoryUrl(t)
	accessToken := acctest.GetAccessToken(t)

	config := fmt.Sprintf(`
		provider "platform" {
			url          = "%s"
			access_token = "%s"
		}

		resource "platform_lifecycle" "project" {
			project_key    = "%s"
			promote_stages = ["DEV"]
		}

		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}

		resource "apptrust_application_version" "%s" {
			application_key  = apptrust_application.%s.application_key
			version          = "%s"
			tag              = "acc-rollback"
			source_artifacts = [{ path = "%s" }]
		}

		resource "apptrust_application_version_promotion" "%s" {
			application_key = apptrust_application_version.%s.application_key
			version         = apptrust_application_version.%s.version
			target_stage    = "%s"
			promotion_type  = "copy"
			depends_on      = [platform_lifecycle.project]
		}

		# Rollback must run after promotion is applied.
		resource "apptrust_application_version_rollback" "%s" {
			application_key = apptrust_application_version.%s.application_key
			version         = apptrust_application_version.%s.version
			from_stage      = "%s"
			depends_on      = [apptrust_application_version_promotion.%s]
		}
	`,
		jfrogURL, accessToken,
		projectKey,
		appName, appKey, appName, projectKey,
		versionName, appName, version, acctest.TestArtifactPath,
		promoName, versionName, versionName, targetStage,
		rollbackName, versionName, versionName, targetStage, promoName,
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"platform": {
				Source:            "jfrog/platform",
				VersionConstraint: "~> 2.2",
			},
		},
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckApplicationVersionDestroy(versionFqrn),
			testAccCheckApplicationDestroy(appFqrn),
		),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rollbackFqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(rollbackFqrn, "version", version),
					resource.TestCheckResourceAttr(rollbackFqrn, "from_stage", targetStage),
					resource.TestCheckResourceAttrSet(rollbackFqrn, "id"),
				),
			},
			{
				ResourceName:      rollbackFqrn,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s:%s:%s", appKey, version, targetStage),
			},
		},
	})
}
