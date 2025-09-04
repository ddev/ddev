package dockerutil

import (
	"errors"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
)

// NetName provides the default network name for ddev.
const NetName = "ddev_default"

// EnsureNetwork will ensure the Docker network for DDEV is created.
func EnsureNetwork(name string, netOptions network.CreateOptions) error {
	// Pre-check for network duplicates
	RemoveNetworkDuplicates(name)

	if !NetExists(name) {
		ctx, client, err := GetDockerClient()
		if err != nil {
			return err
		}
		_, err = client.NetworkCreate(ctx, name, netOptions)
		if err != nil {
			return err
		}
		output.UserOut.Println("Network", name, "created")
	}
	return nil
}

// EnsureDdevNetwork creates or ensures the DDEV network exists or
// exits with fatal.
func EnsureDdevNetwork() {
	// Ensure we have the fallback global DDEV network
	netOptions := network.CreateOptions{
		Driver:   "bridge",
		Internal: false,
		Labels:   map[string]string{"com.ddev.platform": "ddev"},
	}
	err := EnsureNetwork(NetName, netOptions)
	if err != nil {
		output.UserErr.Fatalf("Failed to ensure Docker network %s: %v", NetName, err)
	}
}

// NetworkExists returns true if the named network exists
// Mostly intended for tests
func NetworkExists(netName string) bool {
	return NetExists(strings.ToLower(netName))
}

// NetExists checks to see if the Docker network for DDEV exists.
func NetExists(name string) bool {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return false
	}
	nets, err := client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return false
	}
	for _, n := range nets {
		if n.Name == name {
			return true
		}
	}
	return false
}

// FindNetworksWithLabel returns all networks with the given label
// It ignores the value of the label, is only interested that the label exists.
func FindNetworksWithLabel(label string) ([]network.Inspect, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	nets, err := client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, err
	}

	var matchingNetworks []network.Inspect
	for _, net := range nets {
		if net.Labels != nil {
			if _, exists := net.Labels[label]; exists {
				matchingNetworks = append(matchingNetworks, net)
			}
		}
	}

	return matchingNetworks, nil
}

// RemoveNetwork removes the named Docker network
// netName can also be network's ID
func RemoveNetwork(netName string) error {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return err
	}
	nets, err := client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return err
	}
	// the loop below may not contain such a network
	err = errdefs.NotFound(errors.New("not found"))
	// loop through all nets because there may be duplicates
	// and delete only by ID - it's unique, but the name isn't
	for _, net := range nets {
		if net.Name == netName || net.ID == netName {
			err = client.NetworkRemove(ctx, net.ID)
		}
	}
	return err
}

// RemoveNetworkWithWarningOnError removes the named Docker network
func RemoveNetworkWithWarningOnError(netName string) {
	err := RemoveNetwork(netName)
	// If it's a "no such network" there's no reason to report error
	if err != nil && !IsErrNotFound(err) {
		util.WarningOnce("Unable to remove network %s: %v", netName, err)
	} else if err == nil {
		output.UserOut.Println("Network", netName, "removed")
	}
}

// RemoveNetworkDuplicates removes the duplicates for the named Docker network
// This means that if there is only one network with this name - no action,
// and if there are several such networks, then we leave the first one, and delete the others
func RemoveNetworkDuplicates(netName string) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return
	}
	nets, err := client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return
	}
	networkMatchFound := false
	for _, net := range nets {
		if net.Name == netName || net.ID == netName {
			if networkMatchFound {
				err := client.NetworkRemove(ctx, net.ID)
				// If it's a "no such net" there's no reason to report error
				if err != nil && !IsErrNotFound(err) {
					util.WarningOnce("Unable to remove net %s: %v", netName, err)
				}
			} else {
				networkMatchFound = true
			}
		}
	}
}

// IsErrNotFound returns true if the error is a NotFound error, which is returned
// by the API when some object is not found. It is an alias for [cerrdefs.IsNotFound].
// Used as a wrapper to avoid direct import for docker client.
func IsErrNotFound(err error) bool {
	return cerrdefs.IsNotFound(err)
}
