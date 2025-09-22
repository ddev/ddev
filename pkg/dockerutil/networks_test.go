package dockerutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
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
