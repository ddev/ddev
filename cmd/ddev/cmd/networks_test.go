package cmd

import (
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	dockerNetwork "github.com/docker/docker/api/types/network"
	asrt "github.com/stretchr/testify/assert"
)

// TestNetworkDuplicates makes sure that Docker network duplicates
// are deleted successfully with DDEV
// See https://github.com/ddev/ddev/pull/5508
// Note: duplicate networks cannot be created with Docker >= 25.x.x
// See https://github.com/moby/moby/pull/46251
func TestNetworkDuplicates(t *testing.T) {
	assert := asrt.New(t)

	ctx, client := dockerutil.GetDockerClient()

	// Create two networks with the same name
	networkName := "ddev-" + t.Name() + "_default"

	t.Cleanup(func() {
		err := dockerutil.RemoveNetwork(networkName)
		assert.NoError(err)

		networks, err := client.NetworkList(ctx, dockerNetwork.ListOptions{})
		assert.NoError(err)

		// Ensure the network is not in the list
		for _, network := range networks {
			assert.NotEqual(networkName, network.Name)
		}
	})

	labels := map[string]string{"com.ddev.platform": "ddev"}
	netOptions := dockerNetwork.CreateOptions{
		Driver:   "bridge",
		Internal: false,
		Labels:   labels,
	}

	// Create the first network
	_, err := client.NetworkCreate(ctx, networkName, netOptions)
	assert.NoError(err)

	// Create a second network with the same name
	_, errDuplicate := client.NetworkCreate(ctx, networkName, netOptions)

	// Go library docker/docker/client v25+ throws an error,
	// no matter what version of Docker is installed
	assert.Error(errDuplicate)

	// Check if the network is created
	err = dockerutil.EnsureNetwork(ctx, client, networkName, netOptions)
	assert.NoError(err)

	// This check would fail if there is a network duplicate
	_, err = client.NetworkInspect(ctx, networkName, dockerNetwork.InspectOptions{})
	assert.NoError(err)
}
