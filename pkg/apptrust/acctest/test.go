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
	"strings"
	"sync"
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
	// Ensure the projects the tests reference exist (create-if-missing). AppTrust
	// applications must belong to a pre-existing project, so provision them here
	// instead of requiring out-of-band setup.
	EnsureProjects(t)
}

func GetArtifactoryUrl(t *testing.T) string {
	return testutil.GetEnvVarWithFallback(t, "JFROG_URL", "ARTIFACTORY_URL")
}

func GetAccessToken(t *testing.T) string {
	return testutil.GetEnvVarWithFallback(t, "JFROG_ACCESS_TOKEN", "ARTIFACTORY_ACCESS_TOKEN")
}

// Pre-created project keys for AppTrust application acceptance tests.
// Override via APPTRUST_PROJECT_KEY_1/2/3/4 env vars; defaults to "testproj3038182".
var (
	AppTrustProjectKey1 = getEnvWithDefault("APPTRUST_PROJECT_KEY_1", "testproj3038182")
	AppTrustProjectKey2 = getEnvWithDefault("APPTRUST_PROJECT_KEY_2", "testproj3038182")
	AppTrustProjectKey3 = getEnvWithDefault("APPTRUST_PROJECT_KEY_3", "testproj3038182")
	AppTrustProjectKey4 = getEnvWithDefault("APPTRUST_PROJECT_KEY_4", "testproj3038182")
)

func getEnvWithDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	projectsMu    sync.Mutex
	projectsReady bool
)

// EnsureProjects creates the AppTrust test projects (AppTrustProjectKey1..4) if they
// do not already exist. AppTrust applications must belong to a pre-existing project,
// so acceptance tests provision them here rather than relying on out-of-band setup.
// It is idempotent, deduplicates repeated keys, and only runs once per test binary
// on success (a failure is not cached, so a later test retries).
func EnsureProjects(t *testing.T) {
	t.Helper()

	projectsMu.Lock()
	defer projectsMu.Unlock()
	if projectsReady {
		return
	}

	rc := GetTestResty(t)
	seen := make(map[string]bool)
	for _, key := range []string{AppTrustProjectKey1, AppTrustProjectKey2, AppTrustProjectKey3, AppTrustProjectKey4} {
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		ensureProject(t, rc, key)
	}

	projectsReady = true
}

// ensureProject creates a single project via the Access API, treating an
// already-existing project as success.
func ensureProject(t *testing.T, rc *resty.Client, projectKey string) {
	t.Helper()

	body := map[string]interface{}{
		"project_key":  projectKey,
		"display_name": projectKey,
		"description":  "Terraform provider AppTrust acceptance test project",
		"admin_privileges": map[string]bool{
			"manage_members":   true,
			"manage_resources": true,
			"index_resources":  true,
		},
	}

	resp, err := rc.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post("access/api/v1/projects")
	if err != nil {
		t.Fatalf("EnsureProjects: error creating project %q: %v", projectKey, err)
	}

	switch {
	case resp.IsSuccess():
		return
	case resp.StatusCode() == http.StatusConflict:
		return // already exists
	case strings.Contains(strings.ToLower(resp.String()), "already exists"):
		return
	default:
		t.Fatalf("EnsureProjects: unexpected status %d creating project %q: %s", resp.StatusCode(), projectKey, resp.String())
	}
}

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

// TestArtifactRepo is the generic repository used to stage test artifacts.
// The file at TestArtifactPath must exist — EnsureTestArtifact uploads it if absent.
const (
	TestArtifactRepo = "example-repo-local"
	TestArtifactPath = "example-repo-local/acc-test-artifact.txt"
)

// EnsureTestArtifact uploads a small placeholder file to TestArtifactPath if it does
// not already exist. Promotion tests require a version whose artifact resolved
// successfully (status != FAILED); using a real artifact guarantees this.
func EnsureTestArtifact(t *testing.T) {
	t.Helper()
	rc := GetTestResty(t)
	artifactURL := "artifactory/" + TestArtifactPath
	resp, err := rc.R().Head(artifactURL)
	if err == nil && resp.StatusCode() == http.StatusOK {
		return // already exists
	}
	put, err := rc.R().
		SetHeader("Content-Type", "text/plain").
		SetBody([]byte("acceptance test artifact")).
		Put(artifactURL)
	if err != nil {
		t.Fatalf("EnsureTestArtifact: PUT error: %v", err)
	}
	if put.StatusCode() != http.StatusCreated && put.StatusCode() != http.StatusOK {
		t.Fatalf("EnsureTestArtifact: unexpected status %d: %s", put.StatusCode(), put.String())
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
