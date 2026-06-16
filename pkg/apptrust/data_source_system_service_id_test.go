package apptrust

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSystemServiceIdDataSource_basic(t *testing.T) {
	// TODO: Implement acceptance test
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { /* testAccPreCheck(t) */ },
		ProtoV6ProviderFactories: nil, // TODO: set provider factories
		Steps: []resource.TestStep{
			{
				Config: testAccSystemServiceIdDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// TODO: Add checks
				),
			},
		},
	})
}

func testAccSystemServiceIdDataSourceConfig_basic() string {
	return `
data "apptrust_service_id" "test" {
  jfrog_url = "test-value"
}
`
}
