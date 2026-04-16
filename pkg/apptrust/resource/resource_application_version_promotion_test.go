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

// TestAccApplicationVersionPromotion_basic provisions a lifecycle stage via
// terraform-provider-platform (using the existing global "DEV" stage which already
// has repositories assigned by the platform), creates an application version, and
// promotes it to DEV.
func TestAccApplicationVersionPromotion_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)
	acctest.EnsureTestArtifact(t)

	id, appFqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	versionId, versionFqrn, versionName := testutil.MkNames("test-ver-", "apptrust_application_version")
	_, promoFqrn, promoName := testutil.MkNames("test-promo-", "apptrust_application_version_promotion")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	version := fmt.Sprintf("1.0.%d", versionId)

	// Use the global "DEV" stage. It always exists on JFrog instances and already
	// has the project's application-versions repository assigned. We add it to the
	// project lifecycle so AppTrust can promote to it.
	const targetStage = "DEV"

	jfrogURL := acctest.GetArtifactoryUrl(t)
	accessToken := acctest.GetAccessToken(t)

	config := fmt.Sprintf(`
		provider "platform" {
			url          = "%s"
			access_token = "%s"
		}

		# Register DEV in the project lifecycle so AppTrust can target it.
		# DEV is a global stage that already has the project's repos assigned.
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
			tag              = "acc-test"
			source_artifacts = [{ path = "%s" }]
		}

		resource "apptrust_application_version_promotion" "%s" {
			application_key = apptrust_application_version.%s.application_key
			version         = apptrust_application_version.%s.version
			target_stage    = "%s"
			promotion_type  = "copy"
			depends_on      = [platform_lifecycle.project]
		}
	`,
		jfrogURL, accessToken,
		projectKey,
		appName, appKey, appName, projectKey,
		versionName, appName, version, acctest.TestArtifactPath,
		promoName, versionName, versionName, targetStage,
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
