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
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

func TestAccBoundPackageVersionsDataSource_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, appName := testutil.MkNames("test-app-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)
	dataSourceFqrn := "data.apptrust_bound_package_versions.test"

	// Query bound versions for an application with no bound packages (generic package name)
	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
		data "apptrust_bound_package_versions" "test" {
			application_key = apptrust_application.%s.application_key
			package_type    = "generic"
			package_name    = "no-such-package"
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
					resource.TestCheckResourceAttr(dataSourceFqrn, "package_type", "generic"),
					resource.TestCheckResourceAttr(dataSourceFqrn, "package_name", "no-such-package"),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "versions.#"),
				),
			},
		},
	})
}
