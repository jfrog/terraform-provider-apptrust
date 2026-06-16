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

func TestAccApplicationVersionIntegrationEventResource_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, appFqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	versionId, _, versionName := testutil.MkNames("test-ver-", "apptrust_application_version")
	_, fqrn, name := testutil.MkNames("test-event-", "apptrust_application_version_integration_event")
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
			tag              = "acc-test"
			source_artifacts = [{ path = "generic-repo/readme.md" }]
		}
		resource "apptrust_application_version_integration_event" "%s" {
			application_key = apptrust_application_version.%s.application_key
			version         = apptrust_application_version.%s.version
			reference       = "ext-ref-001"
			status          = "success"
			type            = "scan"
			event_message   = "acc test event"
			properties = {
				source = "terraform-acc"
			}
		}
	`, appName, appKey, appName, projectKey, versionName, appName, version, name, versionName, versionName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(appFqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(fqrn, "version", version),
					resource.TestCheckResourceAttr(fqrn, "reference", "ext-ref-001"),
					resource.TestCheckResourceAttrSet(fqrn, "id"),
				),
			},
		},
	})
}
