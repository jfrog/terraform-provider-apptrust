package apptrust

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLifecycleStagesDataSource_basic(t *testing.T) {
	// TODO: Implement acceptance test
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { /* testAccPreCheck(t) */ },
		ProtoV6ProviderFactories: nil, // TODO: set provider factories
		Steps: []resource.TestStep{
			{
				Config: testAccLifecycleStagesDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// TODO: Add checks
				),
			},
		},
	})
}

func testAccLifecycleStagesDataSourceConfig_basic() string {
	return `
data "apptrust_stages" "test" {
  jfrog_url = "test-value"
  project_key = "test-value"
}
`
}
