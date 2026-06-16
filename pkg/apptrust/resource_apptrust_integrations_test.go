package apptrust

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApptrustIntegrations_basic(t *testing.T) {
	// TODO: Implement acceptance test
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { /* testAccPreCheck(t) */ },
		ProtoV6ProviderFactories: nil, // TODO: set provider factories
		Steps: []resource.TestStep{
			{
				Config: testAccApptrustIntegrationsConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// TODO: Add checks
				),
			},
		},
	})
}

func testAccApptrustIntegrationsConfig_basic() string {
	return `
resource "apptrust_integrations" "test" {
  jfrog_url = "test-value"
  integration_key = "test-value"
}
`
}
