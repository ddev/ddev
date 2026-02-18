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
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
)

// TestNetworkDuplicates makes sure that Docker network duplicates
// are deleted successfully with DDEV
// See https://github.com/ddev/ddev/pull/5508
// Note: duplicate networks cannot be created with Docker >= 25.x.x
// See https://github.com/moby/moby/pull/46251
func TestNetworkDuplicates(t *testing.T) {
	ctx, apiClient, err := dockerutil.GetDockerClient()
	if err != nil {
		t.Fatalf("Could not get docker client: %v", err)
	}

	// Create two networks with the same name
	networkName := "ddev-" + t.Name() + "_default"

	t.Cleanup(func() {
		err := dockerutil.RemoveNetwork(networkName)
		require.NoError(t, err)

		nets, err := apiClient.NetworkList(ctx, client.NetworkListOptions{})
		require.NoError(t, err)

		// Ensure the network is not in the list
		for _, net := range nets.Items {
			require.NotEqual(t, networkName, net.Name)
		}
	})

	labels := map[string]string{"com.ddev.platform": "ddev"}
	netOptions := client.NetworkCreateOptions{
		Driver:   "bridge",
		Internal: false,
		Labels:   labels,
	}

	// Create the first network
	_, err = apiClient.NetworkCreate(ctx, networkName, netOptions)
	require.NoError(t, err)

	// Create a second network with the same name
	_, errDuplicate := apiClient.NetworkCreate(ctx, networkName, netOptions)

	// Go library docker/docker/client v25+ throws an error,
	// no matter what version of Docker is installed
	require.Error(t, errDuplicate)

	// Check if the network is created
	err = dockerutil.EnsureNetwork(networkName, netOptions)
	require.NoError(t, err)

	// This check would fail if there is a network duplicate
	_, err = apiClient.NetworkInspect(ctx, networkName, client.NetworkInspectOptions{})
	require.NoError(t, err)
}

// TestNetworkAmbiguity tests the behavior and setup of Docker networking.
// There should be no crosstalk between different projects
func TestNetworkAmbiguity(t *testing.T) {
	if dockerutil.IsPodman() {
		t.Skip("Skipping because Podman handles networking differently")
	}

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
// when both projects use the same ports (80/443). This validates the fix for #8110:
// the router must be recreated when network aliases change even if ports are identical.
// See pkg/ddevapp/router_compose_template.yaml
// This verifies the functionality of https://docs.docker.com/reference/compose-file/services/#aliases
// Related test: TestInternalAndExternalAccessToURL
func TestNetworkAliases(t *testing.T) {
	runNetworkAliasesTest(t, "80", "443", "80", "443")
}

// TestNetworkAliasesDifferentPorts tests inter-project connectivity using ddev_default
// network aliases when the two projects use different HTTP/HTTPS ports.
// app1 uses ports 8080/8443 and app2 uses the default 80/443.
func TestNetworkAliasesDifferentPorts(t *testing.T) {
	runNetworkAliasesTest(t, "8080", "8443", "80", "443")
}

// runNetworkAliasesTest is the shared implementation for TestNetworkAliases and
// TestNetworkAliasesDifferentPorts. app1HTTPPort/app1HTTPSPort are the router ports
// for app1; app2HTTPPort/app2HTTPSPort are for app2.
func runNetworkAliasesTest(t *testing.T, app1HTTPPort, app1HTTPSPort, app2HTTPPort, app2HTTPSPort string) {
	t.Helper()
	origDir, _ := os.Getwd()

	// Create two temporary projects
	projects := map[string]string{
		t.Name() + "-app1": testcommon.CreateTmpDir(t.Name() + "-app1"),
		t.Name() + "-app2": testcommon.CreateTmpDir(t.Name() + "-app2"),
	}

	// Skip early if HTTPS is not available; this test requires it for network alias connectivity
	for _, projDir := range projects {
		checkApp, err := ddevapp.NewApp(projDir, false)
		require.NoError(t, err)
		if checkApp.CanUseHTTPOnly() {
			t.Skip("Skipping: HTTPS not available")
		}
		break
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)

		out, _ := exec.RunHostCommand(DdevBin, "list")
		t.Logf("\n=========== output of `ddev list` ==========\n%s\n============\n", out)
		out, _ = exec.RunHostCommand("docker", "logs", "ddev-router")
		t.Logf("\n=========== output of `docker logs ddev-router` ==========\n%s\n============\n", out)
		out, _ = exec.RunHostCommand("docker", "inspect", "ddev-router", "--format", "{{json .NetworkSettings.Networks}}")
		t.Logf("\n=========== output of `docker inspect ddev-router --format '{{json .NetworkSettings.Networks}}'` ==========\n%s\n============\n", out)

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

			if strings.Contains(projName, "app1") {
				app.AdditionalHostnames = []string{"api", "admin"}
				app.AdditionalFQDNs = []string{"test1.example.com"}
				app.RouterHTTPPort = app1HTTPPort
				app.RouterHTTPSPort = app1HTTPSPort
			} else {
				app.AdditionalHostnames = []string{"backend", "service"}
				app.AdditionalFQDNs = []string{"test2.example.com"}
				app.RouterHTTPPort = app2HTTPPort
				app.RouterHTTPSPort = app2HTTPSPort
			}

			err = app.WriteConfig()
			require.NoError(t, err)

			// Copy test index.php file
			err = fileutil.CopyFile(filepath.Join(origDir, "testdata", "TestNetworkAliases", "index.php"), filepath.Join(projDir, "index.php"))
			require.NoError(t, err)

			err = app.Start()
			require.NoError(t, err)

			apps = append(apps, app)
		}
	})

	// Get app references after setup
	var app1, app2 *ddevapp.DdevApp
	for _, app := range apps {
		if strings.Contains(app.Name, "app1") {
			app1 = app
		} else {
			app2 = app
		}
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
				httpURL:     app2.GetHTTPURL(),
				httpsURL:    app2.GetHTTPSURL(),
				description: "app1 should be able to reach app2 by project name",
			},
			{
				category:    "project_name",
				name:        "app2_to_app1",
				fromApp:     app2,
				toApp:       app1,
				httpURL:     app1.GetHTTPURL(),
				httpsURL:    app1.GetHTTPSURL(),
				description: "app2 should be able to reach app1 by project name",
			},
			// Additional hostnames tests
			{
				category:    "additional_hostnames",
				name:        "app1_to_app2_backend",
				fromApp:     app1,
				toApp:       app2,
				httpURL:     strings.Replace(app2.GetHTTPURL(), app2.GetHostname(), "backend.ddev.site", 1),
				httpsURL:    strings.Replace(app2.GetHTTPSURL(), app2.GetHostname(), "backend.ddev.site", 1),
				description: "app1 should be able to reach app2 backend hostname",
			},
			{
				category:    "additional_hostnames",
				name:        "app1_to_app2_service",
				fromApp:     app1,
				toApp:       app2,
				httpURL:     strings.Replace(app2.GetHTTPURL(), app2.GetHostname(), "service.ddev.site", 1),
				httpsURL:    strings.Replace(app2.GetHTTPSURL(), app2.GetHostname(), "service.ddev.site", 1),
				description: "app1 should be able to reach app2 service hostname",
			},
			{
				category:    "additional_hostnames",
				name:        "app2_to_app1_api",
				fromApp:     app2,
				toApp:       app1,
				httpURL:     strings.Replace(app1.GetHTTPURL(), app1.GetHostname(), "api.ddev.site", 1),
				httpsURL:    strings.Replace(app1.GetHTTPSURL(), app1.GetHostname(), "api.ddev.site", 1),
				description: "app2 should be able to reach app1 api hostname",
			},
			{
				category:    "additional_hostnames",
				name:        "app2_to_app1_admin",
				fromApp:     app2,
				toApp:       app1,
				httpURL:     strings.Replace(app1.GetHTTPURL(), app1.GetHostname(), "admin.ddev.site", 1),
				httpsURL:    strings.Replace(app1.GetHTTPSURL(), app1.GetHostname(), "admin.ddev.site", 1),
				description: "app2 should be able to reach app1 admin hostname",
			},
			// Additional FQDNs tests
			{
				category:    "additional_fqdns",
				name:        "app1_to_app2",
				fromApp:     app1,
				toApp:       app2,
				httpURL:     strings.Replace(app2.GetHTTPURL(), app2.GetHostname(), "test2.example.com", 1),
				httpsURL:    strings.Replace(app2.GetHTTPSURL(), app2.GetHostname(), "test2.example.com", 1),
				description: "app1 should be able to reach app2 additional FQDNs",
			},
			{
				category:    "additional_fqdns",
				name:        "app2_to_app1",
				fromApp:     app2,
				toApp:       app1,
				httpURL:     strings.Replace(app1.GetHTTPURL(), app1.GetHostname(), "test1.example.com", 1),
				httpsURL:    strings.Replace(app1.GetHTTPSURL(), app1.GetHostname(), "test1.example.com", 1),
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
					"http":  tc.httpURL,
					"https": tc.httpsURL,
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
}
