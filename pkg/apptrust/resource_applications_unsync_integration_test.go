package apptrust

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationsUnsyncIntegration_basic(t *testing.T) {
	// TODO: Implement acceptance test
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { /* testAccPreCheck(t) */ },
		ProtoV6ProviderFactories: nil, // TODO: set provider factories
		Steps: []resource.TestStep{
			{
				Config: testAccApplicationsUnsyncIntegrationConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// TODO: Add checks
				),
			},
		},
	})
}

func testAccApplicationsUnsyncIntegrationConfig_basic() string {
	return `
resource "apptrust_unsync_integration" "test" {
  jfrog_url = "test-value"
  application_key = "test-value"
}
`
}
