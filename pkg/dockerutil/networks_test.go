package dockerutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/require"
)

// TestNetworkDuplicates makes sure that Docker network duplicates
// are deleted successfully with DDEV
// See https://github.com/ddev/ddev/pull/5508
// Note: duplicate networks cannot be created with Docker >= 25.x.x
// See https://github.com/moby/moby/pull/46251
func TestNetworkDuplicates(t *testing.T) {
	ctx, client, err := dockerutil.GetDockerClient()
	if err != nil {
		t.Fatalf("Could not get docker client: %v", err)
	}

	// Create two networks with the same name
	networkName := "ddev-" + t.Name() + "_default"

	t.Cleanup(func() {
		err := dockerutil.RemoveNetwork(networkName)
		require.NoError(t, err)

		nets, err := client.NetworkList(ctx, network.ListOptions{})
		require.NoError(t, err)

		// Ensure the network is not in the list
		for _, net := range nets {
			require.NotEqual(t, networkName, net.Name)
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
	require.NoError(t, err)

	// Create a second network with the same name
	_, errDuplicate := client.NetworkCreate(ctx, networkName, netOptions)

	// Go library docker/docker/client v25+ throws an error,
	// no matter what version of Docker is installed
	require.Error(t, errDuplicate)

	// Check if the network is created
	err = dockerutil.EnsureNetwork(networkName, netOptions)
	require.NoError(t, err)

	// This check would fail if there is a network duplicate
	_, err = client.NetworkInspect(ctx, networkName, network.InspectOptions{})
	require.NoError(t, err)
}

// TestNetworkAmbiguity tests the behavior and setup of Docker networking.
// There should be no crosstalk between different projects
func TestNetworkAmbiguity(t *testing.T) {
	origDir, _ := os.Getwd()

	projects := map[string]string{
		t.Name() + "-app1": testcommon.CreateTmpDir(t.Name() + "-app1"),
		t.Name() + "-app2": testcommon.CreateTmpDir(t.Name() + "-app2"),
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		for projName, projDir := range projects {
			app, err := ddevapp.GetActiveApp(projName)
			if err == nil {
				_ = app.Stop(true, false)
			}
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
			require.NoError(t, err)
		}
		// Create new app
		app, err = ddevapp.NewApp(projDir, false)
		require.NoError(t, err)
		app.Type = nodeps.AppTypePHP
		app.Name = projName
		err = app.WriteConfig()
		require.NoError(t, err)
		err = fileutil.CopyFile(filepath.Join(origDir, "testdata", t.Name(), "docker-compose.test.yaml"), app.GetConfigPath("docker-compose.test.yaml"))
		require.NoError(t, err)
		err = app.Start()
		require.NoError(t, err)
	}

	// With the improved two-network handling, the simple service names
	// are no longer ambiguous. We'll see one entry for web and one for db
	// very ambiguous, but one on web, because it has 'links'
	expectations := map[string]int{"web": 1, "db": 1}
	for projName := range projects {
		app, err := ddevapp.GetActiveApp(projName)
		require.NoError(t, err)
		for c, expectation := range expectations {
			out, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: c,
				Cmd:     "getent hosts test",
			})
			require.NoError(t, err)
			out = strings.Trim(out, "\r\n ")
			ips := strings.Split(out, "\n")
			require.Len(t, ips, expectation)
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

	runTime := util.TimeTrackC(t.Name())

	origDir, _ := os.Getwd()

	// Create two temporary projects
	projects := map[string]string{
		t.Name() + "-app1": testcommon.CreateTmpDir(t.Name() + "-app1"),
		t.Name() + "-app2": testcommon.CreateTmpDir(t.Name() + "-app2"),
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		for projName, projDir := range projects {
			app, err := ddevapp.GetActiveApp(projName)
			if err == nil {
				_ = app.Stop(true, false)
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
				require.NoError(t, err)
			}

			// Create new app
			app, err = ddevapp.NewApp(projDir, false)
			require.NoError(t, err)
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
			require.NoError(t, err)

			// Copy test index.php file
			err = fileutil.CopyFile(filepath.Join(origDir, "testdata", "TestNetworkAliases", "index.php"), filepath.Join(projDir, "index.php"))
			require.NoError(t, err)

			err = app.StartAndWait(5)
			require.NoError(t, err)

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
					"http_web":       "http://ddev-" + tc.toApp.Name + "-web",
					"http_web_alias": tc.httpURL,
				}
				if globalconfig.GetCAROOT() != "" {
					urls["https_web"] = "https://ddev-" + tc.toApp.Name + "-web"
					urls["https_web_alias"] = tc.httpsURL
				}

				for protocol, url := range urls {
					t.Run(tc.name+"_"+protocol, func(t *testing.T) {
						curlCmd := "curl -sS --fail " + url
						out, _, err := tc.fromApp.Exec(&ddevapp.ExecOpts{
							Service: "web",
							Cmd:     curlCmd,
						})
						require.NoError(t, err)
						require.Contains(t, out, "Hello from "+tc.toApp.Name, "Response should contain %s project name (from %s, '%s')", tc.toApp.Name, tc.fromApp.Name, curlCmd)
					})
				}
			}
		})
	}

	out, err := exec.RunHostCommand(DdevBin, "list")
	require.NoError(t, err)
	t.Logf("\n=========== output of ddev list ==========\n%s\n============\n", out)
	out, err = exec.RunHostCommand("docker", "logs", "ddev-router")
	require.NoError(t, err)
	t.Logf("\n=========== output of docker logs ddev-router ==========\n%s\n============\n", out)

	runTime()
}
