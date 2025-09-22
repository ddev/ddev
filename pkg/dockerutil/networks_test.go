package dockerutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/docker/docker/api/types/network"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNetworkDuplicates makes sure that Docker network duplicates
// are deleted successfully with DDEV
// See https://github.com/ddev/ddev/pull/5508
// Note: duplicate networks cannot be created with Docker >= 25.x.x
// See https://github.com/moby/moby/pull/46251
func TestNetworkDuplicates(t *testing.T) {
	assert := asrt.New(t)

	ctx, client, err := dockerutil.GetDockerClient()
	if err != nil {
		t.Fatalf("Could not get docker client: %v", err)
	}

	// Create two networks with the same name
	networkName := "ddev-" + t.Name() + "_default"

	t.Cleanup(func() {
		err := dockerutil.RemoveNetwork(networkName)
		assert.NoError(err)

		nets, err := client.NetworkList(ctx, network.ListOptions{})
		assert.NoError(err)

		// Ensure the network is not in the list
		for _, net := range nets {
			assert.NotEqual(networkName, net.Name)
		}
	})

	labels := map[string]string{"com.ddev.platform": "ddev"}
	netOptions := network.CreateOptions{
		Driver:   "bridge",
		Internal: false,
		Labels:   labels,
	}

	// Create the first network
	_, err = client.NetworkCreate(ctx, networkName, netOptions)
	assert.NoError(err)

	// Create a second network with the same name
	_, errDuplicate := client.NetworkCreate(ctx, networkName, netOptions)

	// Go library docker/docker/client v25+ throws an error,
	// no matter what version of Docker is installed
	assert.Error(errDuplicate)

	// Check if the network is created
	err = dockerutil.EnsureNetwork(networkName, netOptions)
	assert.NoError(err)

	// This check would fail if there is a network duplicate
	_, err = client.NetworkInspect(ctx, networkName, network.InspectOptions{})
	assert.NoError(err)
}

// TestNetworkAmbiguity tests the behavior and setup of Docker networking.
// There should be no crosstalk between different projects
func TestNetworkAmbiguity(t *testing.T) {
	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	projects := map[string]string{
		t.Name() + "-app1": testcommon.CreateTmpDir(t.Name() + "-app1"),
		t.Name() + "-app2": testcommon.CreateTmpDir(t.Name() + "-app2"),
	}
	var err error

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		for projName, projDir := range projects {
			app, err := ddevapp.GetActiveApp(projName)
			assert.NoError(err)
			err = app.Stop(true, false)
			assert.NoError(err)
			_ = os.RemoveAll(projDir)
		}
	})

	// Start a set of projects that contain a simple test container
	// Verify that test is ambiguous or not, with or without `links`
	// Use docker network inspect? Use getent hosts test
	for projName, projDir := range projects {
		// Clean up any existing name conflicts
		app, err := ddevapp.GetActiveApp(projName)
		if err == nil {
			err = app.Stop(true, false)
			assert.NoError(err)
		}
		// Create new app
		app, err = ddevapp.NewApp(projDir, false)
		assert.NoError(err)
		app.Type = nodeps.AppTypePHP
		app.Name = projName
		err = app.WriteConfig()
		assert.NoError(err)
		err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.test.yaml"), app.GetConfigPath("docker-compose.test.yaml"))
		assert.NoError(err)
		err = app.Start()
		assert.NoError(err)
	}

	// With the improved two-network handling, the simple service names
	// are no longer ambiguous. We'll see one entry for web and one for db
	// very ambiguous, but one on web, because it has 'links'
	expectations := map[string]int{"web": 1, "db": 1}
	for projName := range projects {
		app, err := ddevapp.GetActiveApp(projName)
		assert.NoError(err)
		for c, expectation := range expectations {
			out, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: c,
				Cmd:     "getent hosts test",
			})
			require.NoError(t, err)
			out = strings.Trim(out, "\r\n ")
			ips := strings.Split(out, "\n")
			assert.Len(ips, expectation)
		}
	}
}

// TestNetworkAliases tests inter-project connectivity using ddev_default network aliases
// See pkg/ddevapp/router_compose_template.yaml
// This verifies the functionality of https://docs.docker.com/reference/compose-file/services/#aliases
// where projects can communicate with each other via hostnames without external_links
// Related test: TestInternalAndExternalAccessToURL
func TestNetworkAliases(t *testing.T) {
	if nodeps.IsAppleSilicon() || dockerutil.IsColima() || dockerutil.IsLima() || dockerutil.IsRancherDesktop() {
		t.Skip("Skipping on mac Apple Silicon/Lima/Colima/Rancher to ignore problems with 'connection reset by peer'")
	}

	assert := asrt.New(t)

	origDir, _ := os.Getwd()

	// Create two temporary projects
	projects := map[string]string{
		t.Name() + "-app1": testcommon.CreateTmpDir(t.Name() + "-app1"),
		t.Name() + "-app2": testcommon.CreateTmpDir(t.Name() + "-app2"),
	}
	var err error

	t.Cleanup(func() {
		err = os.Chdir(origDir)
		assert.NoError(err)
		for projName, projDir := range projects {
			app, err := ddevapp.GetActiveApp(projName)
			if err == nil {
				err = app.Stop(true, false)
				assert.NoError(err)
			}
			_ = os.RemoveAll(projDir)
		}
	})

	var apps []*ddevapp.DdevApp

	t.Run("setup_projects", func(t *testing.T) {
		// Create and configure both projects
		for projName, projDir := range projects {
			// Clean up any existing name conflicts
			app, err := ddevapp.GetActiveApp(projName)
			if err == nil {
				err = app.Stop(true, false)
				assert.NoError(err)
			}

			// Create new app
			app, err = ddevapp.NewApp(projDir, false)
			assert.NoError(err)
			app.Type = nodeps.AppTypePHP
			app.Name = projName

			// Add different hostnames, FQDNs, and router ports for each project
			if strings.Contains(projName, "app1") {
				app.AdditionalHostnames = []string{"api", "admin"}
				app.AdditionalFQDNs = []string{"test1.example.com"}
				// Use custom router ports for app1
				app.RouterHTTPPort = "8080"
				app.RouterHTTPSPort = "8443"
			} else {
				app.AdditionalHostnames = []string{"backend", "service"}
				app.AdditionalFQDNs = []string{"test2.example.com"}
				// Use default ports (80/443) for app2
				app.RouterHTTPPort = "80"
				app.RouterHTTPSPort = "443"
			}

			err = app.WriteConfig()
			assert.NoError(err)

			// Copy test index.php file
			err = fileutil.CopyFile(filepath.Join(origDir, "testdata", "TestNetworkAliases", "index.php"), filepath.Join(projDir, "index.php"))
			assert.NoError(err)

			err = app.Start()
			assert.NoError(err)

			apps = append(apps, app)
		}
	})

	// Get app references after setup
	var app1, app2 *ddevapp.DdevApp
	if len(apps) >= 2 {
		app1, app2 = apps[0], apps[1]
	}

	// Define test case structure
	type testCase struct {
		category    string
		name        string
		fromApp     *ddevapp.DdevApp
		toApp       *ddevapp.DdevApp
		httpURL     string
		httpsURL    string
		description string
	}

	// Create test case generator function
	createTestCases := func() []testCase {
		return []testCase{
			// Project name tests
			{
				category:    "project_name",
				name:        "app1_to_app2",
				fromApp:     app1,
				toApp:       app2,
				httpURL:     "http://" + app2.GetHostname(),
				httpsURL:    "https://" + app2.GetHostname(),
				description: "app1 should be able to reach app2 by project name",
			},
			{
				category:    "project_name",
				name:        "app2_to_app1",
				fromApp:     app2,
				toApp:       app1,
				httpURL:     "http://" + app1.GetHostname() + ":8080",
				httpsURL:    "https://" + app1.GetHostname() + ":8443",
				description: "app2 should be able to reach app1 by project name",
			},
			// Additional hostnames tests
			{
				category:    "additional_hostnames",
				name:        "app1_to_app2_backend",
				fromApp:     app1,
				toApp:       app2,
				httpURL:     "http://backend.ddev.site",
				httpsURL:    "https://backend.ddev.site",
				description: "app1 should be able to reach app2 backend hostname",
			},
			{
				category:    "additional_hostnames",
				name:        "app1_to_app2_service",
				fromApp:     app1,
				toApp:       app2,
				httpURL:     "http://service.ddev.site",
				httpsURL:    "https://service.ddev.site",
				description: "app1 should be able to reach app2 service hostname",
			},
			{
				category:    "additional_hostnames",
				name:        "app2_to_app1_api",
				fromApp:     app2,
				toApp:       app1,
				httpURL:     "http://api.ddev.site:8080",
				httpsURL:    "https://api.ddev.site:8443",
				description: "app2 should be able to reach app1 api hostname",
			},
			{
				category:    "additional_hostnames",
				name:        "app2_to_app1_admin",
				fromApp:     app2,
				toApp:       app1,
				httpURL:     "http://admin.ddev.site:8080",
				httpsURL:    "https://admin.ddev.site:8443",
				description: "app2 should be able to reach app1 admin hostname",
			},
			// Additional FQDNs tests
			{
				category:    "additional_fqdns",
				name:        "app1_to_app2",
				fromApp:     app1,
				toApp:       app2,
				httpURL:     "http://test2.example.com",
				httpsURL:    "https://test2.example.com",
				description: "app1 should be able to reach app2 additional FQDNs",
			},
			{
				category:    "additional_fqdns",
				name:        "app2_to_app1",
				fromApp:     app2,
				toApp:       app1,
				httpURL:     "http://test1.example.com:8080",
				httpsURL:    "https://test1.example.com:8443",
				description: "app2 should be able to reach app1 additional FQDNs",
			},
		}
	}

	// Group tests by category and run them
	allTestCases := createTestCases()
	categories := []string{"project_name", "additional_hostnames", "additional_fqdns"}
	for _, category := range categories {
		t.Run(category, func(t *testing.T) {
			for _, tc := range allTestCases {
				if tc.category != category {
					continue
				}
				// Test both HTTP and HTTPS URLs
				urls := map[string]string{
					"http": tc.httpURL,
				}
				if globalconfig.GetCAROOT() != "" {
					urls["https"] = tc.httpsURL
				}

				for protocol, url := range urls {
					t.Run(tc.name+"_"+protocol, func(t *testing.T) {
						curlCmd := "curl -sS --fail " + url
						out, _, err := tc.fromApp.Exec(&ddevapp.ExecOpts{
							Service: "web",
							Cmd:     curlCmd,
						})
						assert.NoError(err)
						assert.Contains(out, "Hello from "+tc.toApp.Name, "Response should contain %s project name (from %s, '%s')", tc.toApp.Name, tc.fromApp.Name, curlCmd)
					})
				}
			}
		})
	}
}
