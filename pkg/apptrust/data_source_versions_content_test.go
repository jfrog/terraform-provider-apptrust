package apptrust

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVersionsContentDataSource_basic(t *testing.T) {
	// TODO: Implement acceptance test
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { /* testAccPreCheck(t) */ },
		ProtoV6ProviderFactories: nil, // TODO: set provider factories
		Steps: []resource.TestStep{
			{
				Config: testAccVersionsContentDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// TODO: Add checks
				),
			},
		},
	})
}

func testAccVersionsContentDataSourceConfig_basic() string {
	return `
data "apptrust_content" "test" {
  jfrog_url = "test-value"
  application_key = "test-value"
  version = "test-value"
}
`
}
