package cmd

import (
	"testing"

	"github.com/ddev/ddev/pkg/dockerutil"
	docker "github.com/fsouza/go-dockerclient"
	asrt "github.com/stretchr/testify/assert"
)

// TestNetworkDuplicates makes sure that Docker network duplicates
// are deleted successfully with DDEV
// See https://github.com/ddev/ddev/pull/5508
// Note: duplicate networks cannot be created with Docker >= 25.x.x
// See https://github.com/moby/moby/pull/46251
func TestNetworkDuplicates(t *testing.T) {
	assert := asrt.New(t)

	client := dockerutil.GetDockerClient()

	// Create two networks with the same name
	networkName := "ddev-" + t.Name() + "_default"

	t.Cleanup(func() {
		err := dockerutil.RemoveNetwork(networkName)
		assert.NoError(err)

		networks, err := client.ListNetworks()
		assert.NoError(err)

		// Ensure the network is not in the list
		for _, network := range networks {
			assert.NotEqual(networkName, network.Name)
		}
	})

	labels := map[string]string{"com.ddev.platform": "ddev"}
	netOptions := docker.CreateNetworkOptions{
		Name:     networkName,
		Driver:   "bridge",
		Internal: false,
		Labels:   labels,
	}

	// Create the first network
	_, err := client.CreateNetwork(netOptions)
	assert.NoError(err)

	// Create a second network with the same name
	_, errDuplicate := client.CreateNetwork(netOptions)

	errVersion := dockerutil.CheckDockerVersion(">= 25.0.0-alpha1")

	if errVersion == nil {
		// Duplicate cannot be created with Docker >= 25.x.x
		assert.Error(errDuplicate)
	} else {
		// Duplicate can be created with Docker < 25.x.x
		assert.NoError(errDuplicate)
	}

	// The duplicate network is removed here
	err = dockerutil.EnsureNetwork(client, networkName, netOptions)
	assert.NoError(err)

	// This check would fail if there is a network duplicate
	_, err = client.NetworkInfo(networkName)
	assert.NoError(err)
}
