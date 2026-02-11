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
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

const applicationEndpoint = "apptrust/api/v1/applications"

func TestAccApplication_basic(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1
	appKey := fmt.Sprintf("app-%d", id)

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, appKey, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", appKey),
					resource.TestCheckResourceAttr(fqrn, "application_name", name),
					resource.TestCheckResourceAttr(fqrn, "project_key", projectKey),
				),
			},
			{
				ResourceName:      fqrn,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccApplication_full(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-full-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = "A comprehensive test application"
			maturity_level   = "production"
			criticality      = "high"
			
			labels = {
				environment = "production"
				region      = "us-east-1"
				team        = "platform"
			}
			
			user_owners = ["admin"]
			group_owners = ["readers"]
		}
	`, name, id, name, projectKey)

	updatedConfig := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s Updated"
			project_key      = "%s"
			description      = "An updated comprehensive test application"
			maturity_level   = "experimental"
			criticality      = "critical"
			
			labels = {
				environment = "staging"
				region      = "us-west-2"
				team        = "devops"
				new_label   = "new_value"
			}
			
			user_owners = ["admin", "test-user"]
			group_owners = ["readers", "developers"]
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					resource.TestCheckResourceAttr(fqrn, "application_name", name),
					resource.TestCheckResourceAttr(fqrn, "project_key", projectKey),
					resource.TestCheckResourceAttr(fqrn, "description", "A comprehensive test application"),
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "production"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "high"),
					resource.TestCheckResourceAttr(fqrn, "labels.environment", "production"),
					resource.TestCheckResourceAttr(fqrn, "labels.region", "us-east-1"),
					resource.TestCheckResourceAttr(fqrn, "labels.team", "platform"),
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "user_owners.0", "admin"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.0", "readers"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					resource.TestCheckResourceAttr(fqrn, "application_name", name+" Updated"),
					resource.TestCheckResourceAttr(fqrn, "description", "An updated comprehensive test application"),
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "experimental"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "critical"),
					resource.TestCheckResourceAttr(fqrn, "labels.environment", "staging"),
					resource.TestCheckResourceAttr(fqrn, "labels.region", "us-west-2"),
					resource.TestCheckResourceAttr(fqrn, "labels.team", "devops"),
					resource.TestCheckResourceAttr(fqrn, "labels.new_label", "new_value"),
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "2"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_name", name),
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "production"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "high"),
				),
			},
		},
	})
}

func TestAccApplication_minimal(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-min-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					resource.TestCheckResourceAttr(fqrn, "application_name", name),
					resource.TestCheckResourceAttr(fqrn, "project_key", projectKey),
					resource.TestCheckNoResourceAttr(fqrn, "description"),
					// API returns "unspecified" by default for maturity_level and criticality
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "unspecified"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "unspecified"),
					resource.TestCheckNoResourceAttr(fqrn, "labels"),
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
		},
	})
}

func TestAccApplication_updateFields(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-update-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	config1 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, id, name, projectKey)

	config2 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = "Added description"
		}
	`, name, id, name, projectKey)

	config3 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = "Updated description"
			maturity_level   = "production"
			criticality      = "medium"
		}
	`, name, id, name, projectKey)

	config4 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			maturity_level   = "end_of_life"
			criticality      = "low"
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "description"),
					// API returns "unspecified" by default for maturity_level and criticality
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "unspecified"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "unspecified"),
				),
			},
			{
				Config: config2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "description", "Added description"),
					// API returns "unspecified" by default for maturity_level and criticality
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "unspecified"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "unspecified"),
				),
			},
			{
				Config: config3,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "description", "Updated description"),
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "production"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "medium"),
				),
			},
			{
				Config: config4,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "description"),
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "end_of_life"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "low"),
				),
			},
		},
	})
}

func TestAccApplication_labels(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-labels-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	config1 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, id, name, projectKey)

	config2 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			labels = {
				key1 = "value1"
			}
		}
	`, name, id, name, projectKey)

	config3 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			labels = {
				key1 = "value1"
				key2 = "value2"
				key3 = "value3"
			}
		}
	`, name, id, name, projectKey)

	config4 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			labels = {
				key2 = "updated_value2"
			}
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "labels"),
				),
			},
			{
				Config: config2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "labels.key1", "value1"),
				),
			},
			{
				Config: config3,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "labels.key1", "value1"),
					resource.TestCheckResourceAttr(fqrn, "labels.key2", "value2"),
					resource.TestCheckResourceAttr(fqrn, "labels.key3", "value3"),
				),
			},
			{
				Config: config4,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "labels.key1"),
					resource.TestCheckResourceAttr(fqrn, "labels.key2", "updated_value2"),
					resource.TestCheckNoResourceAttr(fqrn, "labels.key3"),
				),
			},
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "labels"),
				),
			},
		},
	})
}

func TestAccApplication_owners(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-owners-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	config1 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, id, name, projectKey)

	config2 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			user_owners = ["admin"]
		}
	`, name, id, name, projectKey)

	config3 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			user_owners = ["admin"]
			group_owners = ["readers"]
		}
	`, name, id, name, projectKey)

	config4 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			user_owners = ["admin", "test-user"]
			group_owners = ["readers", "developers"]
		}
	`, name, id, name, projectKey)

	config5 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			group_owners = ["readers"]
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
			{
				Config: config2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "user_owners.0", "admin"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
			{
				Config: config3,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.0", "readers"),
				),
			},
			{
				Config: config4,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "2"),
				),
			},
			{
				Config: config5,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "1"),
				),
			},
		},
	})
}

func TestAccApplication_maturityLevels(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	projectKey := acctest.AppTrustProjectKey1

	testCases := []struct {
		name          string
		maturityLevel string
	}{
		{"unspecified", "unspecified"},
		{"experimental", "experimental"},
		{"production", "production"},
		{"end_of_life", "end_of_life"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, fqrn, name := testutil.MkNames(fmt.Sprintf("test-app-%s-", tc.name), "apptrust_application")

			config := fmt.Sprintf(`
				resource "apptrust_application" "%s" {
					application_key  = "app-%d"
					application_name = "%s"
					project_key      = "%s"
					maturity_level   = "%s"
				}
			`, name, id, name, projectKey, tc.maturityLevel)

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				PreCheck:                 func() { acctest.PreCheck(t) },
				CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(fqrn, "maturity_level", tc.maturityLevel),
						),
					},
				},
			})
		})
	}
}

func TestAccApplication_criticalityLevels(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	projectKey := acctest.AppTrustProjectKey1

	testCases := []struct {
		name        string
		criticality string
	}{
		{"unspecified", "unspecified"},
		{"low", "low"},
		{"medium", "medium"},
		{"high", "high"},
		{"critical", "critical"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, fqrn, name := testutil.MkNames(fmt.Sprintf("test-app-%s-", tc.name), "apptrust_application")

			config := fmt.Sprintf(`
				resource "apptrust_application" "%s" {
					application_key  = "app-%d"
					application_name = "%s"
					project_key      = "%s"
					criticality      = "%s"
				}
			`, name, id, name, projectKey, tc.criticality)

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				PreCheck:                 func() { acctest.PreCheck(t) },
				CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(fqrn, "criticality", tc.criticality),
						),
					},
				},
			})
		})
	}
}

func TestAccApplication_applicationKeyBoundaries(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	projectKey := acctest.AppTrustProjectKey1

	// Test minimum length (2 characters)
	t.Run("min_length", func(t *testing.T) {
		id, fqrn, name := testutil.MkNames("ab", "apptrust_application")
		config := fmt.Sprintf(`
			resource "apptrust_application" "%s" {
				application_key  = "app-%d"
				application_name = "%s"
				project_key      = "%s"
			}
		`, name, id, name, projectKey)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
			PreCheck:                 func() { acctest.PreCheck(t) },
			CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					),
				},
			},
		})
	})

	// Test single character (should be valid per regex)
	t.Run("single_char", func(t *testing.T) {
		id, fqrn, name := testutil.MkNames("a", "apptrust_application")
		config := fmt.Sprintf(`
			resource "apptrust_application" "%s" {
				application_key  = "app-%d"
				application_name = "%s"
				project_key      = "%s"
			}
		`, name, id, name, projectKey)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
			PreCheck:                 func() { acctest.PreCheck(t) },
			CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					),
				},
			},
		})
	})

	// Test with hyphens
	t.Run("with_hyphens", func(t *testing.T) {
		id, fqrn, name := testutil.MkNames("test-app-with-hyphens", "apptrust_application")
		config := fmt.Sprintf(`
			resource "apptrust_application" "%s" {
				application_key  = "app-%d"
				application_name = "%s"
				project_key      = "%s"
			}
		`, name, id, name, projectKey)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
			PreCheck:                 func() { acctest.PreCheck(t) },
			CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					),
				},
			},
		})
	})
}

func TestAccApplication_applicationNameBoundaries(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	projectKey := acctest.AppTrustProjectKey1

	// Test minimum length (1 character)
	t.Run("min_length", func(t *testing.T) {
		id, fqrn, name := testutil.MkNames("a", "apptrust_application")
		config := fmt.Sprintf(`
			resource "apptrust_application" "%s" {
				application_key  = "app-%d"
				application_name = "%s"
				project_key      = "%s"
			}
		`, name, id, name, projectKey)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
			PreCheck:                 func() { acctest.PreCheck(t) },
			CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(fqrn, "application_name", name),
					),
				},
			},
		})
	})

	// Test with spaces, hyphens, and underscores in application_name (not resource name)
	t.Run("special_chars", func(t *testing.T) {
		id, fqrn, name := testutil.MkNames("test-app-special", "apptrust_application")
		appName := "test-app_with spaces and-hyphens"
		config := fmt.Sprintf(`
			resource "apptrust_application" "%s" {
				application_key  = "app-%d"
				application_name = "%s"
				project_key      = "%s"
			}
		`, name, id, appName, projectKey)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
			PreCheck:                 func() { acctest.PreCheck(t) },
			CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(fqrn, "application_name", appName),
					),
				},
			},
		})
	})
}

func TestAccApplication_planChecks(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-plan-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = "Test application"
			maturity_level   = "production"
			criticality      = "high"
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
				),
			},
			{
				Config:             config,
				ExpectNonEmptyPlan: false,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccApplication_import(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-import-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = "Test import"
			maturity_level   = "production"
			criticality      = "high"
			labels = {
				test = "import"
			}
			user_owners = ["admin"]
			group_owners = ["readers"]
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:            fqrn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{}, // All fields should be importable
			},
		},
	})
}

func TestAccApplication_unspecifiedValues(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-unspec-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	// Test that "unspecified" values are kept in state (API returns "unspecified")
	config1 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			maturity_level   = "unspecified"
			criticality      = "unspecified"
		}
	`, name, id, name, projectKey)

	config2 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					// API returns "unspecified" values, so they should be in state
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "unspecified"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "unspecified"),
				),
			},
			{
				Config: config2,
				Check: resource.ComposeTestCheckFunc(
					// When not set, API returns "unspecified" as default
					resource.TestCheckResourceAttr(fqrn, "maturity_level", "unspecified"),
					resource.TestCheckResourceAttr(fqrn, "criticality", "unspecified"),
				),
			},
		},
	})
}

func TestAccApplication_emptyLists(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-empty-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	// Empty list []: when API omits response, state preserves empty list
	configEmptyList := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			user_owners      = []
			group_owners     = []
		}
	`, name, id, name, projectKey)

	// Explicit null: when API omits response, state keeps null
	configNull := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			user_owners      = null
			group_owners     = null
		}
	`, name, id, name, projectKey)

	// No value (omitted)
	configOmitted := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, id, name, projectKey)

	configWithValues := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			user_owners      = ["admin"]
			group_owners     = ["readers"]
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: configEmptyList,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "0"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "0"),
				),
			},
			{
				Config: configWithValues,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "1"),
				),
			},
			{
				Config: configEmptyList,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "0"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "0"),
				),
			},
			{
				Config: configNull,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
			{
				Config: configWithValues,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "1"),
				),
			},
			{
				Config: configOmitted,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
		},
	})
}

// TestAccApplication_nullAndOmittedValues verifies that optional fields behave correctly
// when omitted (not in config), set to null, set to empty list/map, or have values.
func TestAccApplication_nullAndOmittedValues(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-null-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	// Step 1: No optional values (all omitted)
	configOmitted := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, id, name, projectKey)

	// Step 2: Explicit null for all optional fields
	configExplicitNull := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = null
			labels           = null
			user_owners      = null
			group_owners     = null
		}
	`, name, id, name, projectKey)

	// Step 3: Set values for all optional fields
	configWithValues := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = "test description"
			labels = {
				env = "test"
			}
			user_owners  = ["admin"]
			group_owners = ["readers"]
		}
	`, name, id, name, projectKey)

	// Step 4: Clear back to null (explicit null)
	configClearToNull := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = null
			labels           = null
			user_owners      = null
			group_owners     = null
		}
	`, name, id, name, projectKey)

	// Step 5: user_owners/group_owners as null (same as explicit null, verify no drift)
	configOwnersNull := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			user_owners      = null
			group_owners     = null
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: configOmitted,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					resource.TestCheckResourceAttr(fqrn, "application_name", name),
					resource.TestCheckNoResourceAttr(fqrn, "description"),
					resource.TestCheckNoResourceAttr(fqrn, "labels"),
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
			{
				Config: configExplicitNull,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					// Explicit null: optional attrs may be absent or null in state
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
			{
				Config: configWithValues,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "description", "test description"),
					resource.TestCheckResourceAttr(fqrn, "labels.env", "test"),
					resource.TestCheckResourceAttr(fqrn, "user_owners.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "user_owners.0", "admin"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "group_owners.0", "readers"),
				),
			},
			{
				Config: configClearToNull,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					// After clearing to null, owners should be absent or empty
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
			{
				Config: configOmitted,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					resource.TestCheckNoResourceAttr(fqrn, "description"),
					resource.TestCheckNoResourceAttr(fqrn, "labels"),
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
			{
				Config: configOwnersNull,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
					resource.TestCheckNoResourceAttr(fqrn, "user_owners"),
					resource.TestCheckNoResourceAttr(fqrn, "group_owners"),
				),
			},
		},
	})
}

// TestAccApplication_forceReplace verifies that changing application_key or project_key
// triggers resource replacement (DestroyBeforeCreate) and state is correct after apply.
func TestAccApplication_forceReplace(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-replace-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	key1 := fmt.Sprintf("app-replace-%d", id)
	key2 := fmt.Sprintf("app-replaced-%d", id)

	config1 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, key1, name, projectKey)

	config2 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "%s"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, key2, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", key1),
					resource.TestCheckResourceAttr(fqrn, "id", key1),
				),
			},
			{
				Config: config2,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(fqrn, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "application_key", key2),
					resource.TestCheckResourceAttr(fqrn, "id", key2),
				),
			},
		},
	})
}

// TestAccApplication_emptyDescription verifies empty string and clearing description.
func TestAccApplication_emptyDescription(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-desc-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	configWithDesc := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = "some description"
		}
	`, name, id, name, projectKey)

	configEmptyDesc := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = ""
		}
	`, name, id, name, projectKey)

	configNullDesc := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			description      = null
		}
	`, name, id, name, projectKey)

	configOmitted := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: configWithDesc,
				Check:  resource.TestCheckResourceAttr(fqrn, "description", "some description"),
			},
			{
				Config: configEmptyDesc,
				Check: resource.ComposeTestCheckFunc(
					// Provider preserves empty string in state when plan had description = ""
					resource.TestCheckResourceAttr(fqrn, "description", ""),
					resource.TestCheckResourceAttr(fqrn, "application_key", fmt.Sprintf("app-%d", id)),
				),
			},
			{
				Config: configWithDesc,
				Check:  resource.TestCheckResourceAttr(fqrn, "description", "some description"),
			},
			{
				Config: configNullDesc,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "description"),
				),
			},
			{
				Config: configOmitted,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "description"),
				),
			},
		},
	})
}

// TestAccApplication_emptyLabelsMap verifies that setting labels = {} clears labels on update.
func TestAccApplication_emptyLabelsMap(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-labels-empty-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	configWithLabels := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			labels = {
				env = "test"
			}
		}
	`, name, id, name, projectKey)

	configEmptyMap := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
			labels = {}
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: configWithLabels,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "labels.env", "test"),
				),
			},
			{
				Config: configEmptyMap,
				Check: resource.ComposeTestCheckFunc(
					// Provider preserves empty map in state; no label keys should exist
					resource.TestCheckNoResourceAttr(fqrn, "labels.env"),
					resource.TestCheckResourceAttr(fqrn, "labels.%", "0"),
				),
			},
		},
	})
}

// TestAccApplication_importMinimal verifies import when the remote application has minimal/no optional fields.
func TestAccApplication_importMinimal(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn, name := testutil.MkNames("test-app-import-min-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	config := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name, id, name, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      fqrn,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccApplication_conflict(t *testing.T) {
	acctest.SkipIfNotAcc(t)
	acctest.PreCheck(t)

	id, fqrn1, name1 := testutil.MkNames("test-app-conflict-", "apptrust_application")
	_, _, name2 := testutil.MkNames("test-app-conflict-2-", "apptrust_application")
	projectKey := acctest.AppTrustProjectKey1

	config1 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name1, id, name1, projectKey)

	// Try to create another application with the same key (should fail with conflict)
	config2 := fmt.Sprintf(`
		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}

		resource "apptrust_application" "%s" {
			application_key  = "app-%d"
			application_name = "%s"
			project_key      = "%s"
		}
	`, name1, id, name1, projectKey, name2, id, name2, projectKey)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             testAccCheckApplicationDestroy(fqrn1),
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn1, "application_key", fmt.Sprintf("app-%d", id)),
				),
			},
			{
				Config:      config2,
				ExpectError: regexp.MustCompile(`Application Already Exists|already exists|conflict`),
			},
		},
	})
}

func testAccCheckApplicationDestroy(id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("error: Resource id [%s] not found", id)
		}

		applicationKey := rs.Primary.Attributes["application_key"]
		if applicationKey == "" {
			return fmt.Errorf("error: application_key not found in state")
		}

		client, err := acctest.GetTestRestyFromEnv()
		if err != nil {
			return fmt.Errorf("error creating resty client: %w", err)
		}

		// First, check if the application still exists
		response, err := client.R().
			SetPathParam("application_key", applicationKey).
			Get(applicationEndpoint + "/{application_key}")

		if err != nil {
			return err
		}

		// If application doesn't exist, we're done
		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		// If there was an error checking, return it
		if response.IsError() {
			return fmt.Errorf("error checking application: %s", response.String())
		}

		// Application still exists - attempt to delete it for cleanup
		deleteResponse, err := client.R().
			SetPathParam("application_key", applicationKey).
			Delete(applicationEndpoint + "/{application_key}")

		if err != nil {
			return fmt.Errorf("error deleting application %s during cleanup: %w", applicationKey, err)
		}

		// Check if deletion was successful (204 No Content or 200 OK are both acceptable)
		if deleteResponse.StatusCode() == http.StatusNotFound {
			// Already deleted, that's fine
			return nil
		}

		if deleteResponse.IsError() {
			// If deletion failed, return error
			return fmt.Errorf("error: application %s still exists and could not be deleted during cleanup: %s", applicationKey, deleteResponse.String())
		}

		// Deletion successful (200 or 204)
		return nil
	}
}
