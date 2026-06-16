package apptrust

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVersionsPromotionsDataSource_basic(t *testing.T) {
	// TODO: Implement acceptance test
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { /* testAccPreCheck(t) */ },
		ProtoV6ProviderFactories: nil, // TODO: set provider factories
		Steps: []resource.TestStep{
			{
				Config: testAccVersionsPromotionsDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// TODO: Add checks
				),
			},
		},
	})
}

func testAccVersionsPromotionsDataSourceConfig_basic() string {
	return `
data "apptrust_promotions" "test" {
  jfrog_url = "test-value"
  application_key = "test-value"
  version = "test-value"
  created_millis = "test-value"
}
`
}
