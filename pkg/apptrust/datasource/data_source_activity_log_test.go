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
)

func TestAccActivityLogDataSource_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	dataSourceFqrn := "data.apptrust_activity_log.test"

	config := `
		data "apptrust_activity_log" "test" {
			limit = 10
		}
	`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					// List attribute is checked by count, not by value.
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "activity_logs.#"),
				),
			},
		},
	})
}

func TestAccActivityLogDataSource_withFiltersAndSort(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	dataSourceFqrn := "data.apptrust_activity_log.test"

	config := `
		data "apptrust_activity_log" "test" {
			result   = ["success"]
			sort_by  = "timestamp"
			sort     = "desc"
			limit    = 25
			offset   = 0
		}
	`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "total"),
					resource.TestCheckResourceAttrSet(dataSourceFqrn, "activity_logs.#"),
					testAccCheckActivityLogPageSize(dataSourceFqrn, 25),
				),
			},
		},
	})
}

// testAccCheckActivityLogPageSize verifies activity_logs count does not exceed the requested limit.
func testAccCheckActivityLogPageSize(fqrn string, maxCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[fqrn]
		if !ok {
			return fmt.Errorf("datasource not found: %s", fqrn)
		}
		countStr, ok := rs.Primary.Attributes["activity_logs.#"]
		if !ok {
			return fmt.Errorf("activity_logs.# not found for %s", fqrn)
		}
		var count int
		if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil {
			return fmt.Errorf("activity_logs.# invalid: %q", countStr)
		}
		if count > maxCount {
			return fmt.Errorf("activity_logs.# = %d exceeds limit %d", count, maxCount)
		}
		return nil
	}
}
