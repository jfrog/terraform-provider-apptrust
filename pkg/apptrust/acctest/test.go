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

package acctest

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	provider "github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/provider"
	"github.com/jfrog/terraform-provider-shared/client"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

// ProtoV6ProviderFactories is used to instantiate the Framework provider
// during acceptance tests.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"apptrust": providerserver.NewProtocol6WithError(provider.Framework()()),
}

// PreCheck This function should be present in every acceptance test.
func PreCheck(t *testing.T) {
	// Verify required environment variables are set
	_ = GetArtifactoryUrl(t)
	_ = GetAccessToken(t)
}

func GetArtifactoryUrl(t *testing.T) string {
	return testutil.GetEnvVarWithFallback(t, "JFROG_URL", "ARTIFACTORY_URL")
}

func GetAccessToken(t *testing.T) string {
	return testutil.GetEnvVarWithFallback(t, "JFROG_ACCESS_TOKEN", "ARTIFACTORY_ACCESS_TOKEN")
}

// Pre-created project keys for AppTrust application acceptance tests.
// Projects aa, bb, cc, dd must exist in the test environment.
const (
	AppTrustProjectKey1 = "aa"
	AppTrustProjectKey2 = "bb"
	AppTrustProjectKey3 = "cc"
	AppTrustProjectKey4 = "dd"
)

func GetTestResty(t *testing.T) *resty.Client {
	artifactoryUrl := GetArtifactoryUrl(t)
	restyClient, err := client.Build(artifactoryUrl, "")
	if err != nil {
		t.Fatal(err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	restyClient.SetTLSClientConfig(tlsConfig)
	restyClient.SetRetryCount(5)
	accessToken := GetAccessToken(t)
	restyClient, err = client.AddAuth(restyClient, "", accessToken)
	if err != nil {
		t.Fatal(err)
	}
	return restyClient
}

// GetTestRestyFromEnv builds a resty client from environment variables without requiring testing.T
// This is useful for CheckDestroy functions that don't have access to testing.T
func GetTestRestyFromEnv() (*resty.Client, error) {
	artifactoryUrl := testutil.GetEnvVarWithFallback(nil, "JFROG_URL", "ARTIFACTORY_URL")
	if artifactoryUrl == "" {
		return nil, fmt.Errorf("JFROG_URL or ARTIFACTORY_URL environment variable must be set")
	}

	restyClient, err := client.Build(artifactoryUrl, "")
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	restyClient.SetTLSClientConfig(tlsConfig)
	restyClient.SetRetryCount(5)
	accessToken := testutil.GetEnvVarWithFallback(nil, "JFROG_ACCESS_TOKEN", "ARTIFACTORY_ACCESS_TOKEN")
	if accessToken == "" {
		return nil, fmt.Errorf("JFROG_ACCESS_TOKEN or ARTIFACTORY_ACCESS_TOKEN environment variable must be set")
	}
	restyClient, err = client.AddAuth(restyClient, "", accessToken)
	if err != nil {
		return nil, err
	}
	return restyClient, nil
}

// SkipIfNotAcc skips the test if TF_ACC is not set
func SkipIfNotAcc(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Skipping acceptance test. Set TF_ACC=1 to run.")
	}
}

// TestAccCheckApplicationDestroy checks if an application resource has been destroyed.
func TestAccCheckApplicationDestroy(fqrn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		restyClient, err := GetTestRestyFromEnv()
		if err != nil {
			return err
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "apptrust_application" {
				continue
			}

			response, err := restyClient.R().
				SetPathParam("application_key", rs.Primary.ID).
				Get("apptrust/api/v1/applications/{application_key}")
			if err != nil {
				return err
			}

			if response.StatusCode() == http.StatusNotFound {
				return nil
			}

			if response.IsSuccess() {
				return fmt.Errorf("application %s still exists", rs.Primary.ID)
			}
		}

		return nil
	}
}
