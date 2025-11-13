package dockerutil

import (
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// NetName provides the default network name for ddev.
const NetName = "ddev_default"

// EnsureNetwork will ensure the Docker network for DDEV is created.
func EnsureNetwork(name string, netOptions client.NetworkCreateOptions) error {
	// Pre-check for network duplicates
	RemoveNetworkDuplicates(name)

	if !NetExists(name) {
		ctx, apiClient, err := GetDockerClient()
		if err != nil {
			return err
		}
		_, err = apiClient.NetworkCreate(ctx, name, netOptions)
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
	netOptions := client.NetworkCreateOptions{
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
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return false
	}
	nets, err := apiClient.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return false
	}
	for _, n := range nets.Items {
		if n.Name == name {
			return true
		}
	}
	return false
}

// FindNetworksWithLabel returns all networks with the given label
// It ignores the value of the label, is only interested that the label exists.
func FindNetworksWithLabel(label string) ([]network.Summary, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	nets, err := apiClient.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	var matchingNetworks []network.Summary
	for _, net := range nets.Items {
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
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return err
	}
	nets, err := apiClient.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return err
	}
	// the loop below may not contain such a network
	err = cerrdefs.ErrNotFound
	// loop through all networks because there may be duplicates
	// and delete only by ID - it's unique, but the name isn't
	for _, net := range nets.Items {
		if net.Name == netName || net.ID == netName {
			_, err = apiClient.NetworkRemove(ctx, net.ID, client.NetworkRemoveOptions{})
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
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return
	}
	nets, err := apiClient.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return
	}
	networkMatchFound := false
	for _, net := range nets.Items {
		if net.Name == netName || net.ID == netName {
			if networkMatchFound {
				_, err := apiClient.NetworkRemove(ctx, net.ID, client.NetworkRemoveOptions{})
				// If it's a "no such network" there's no reason to report error
				if err != nil && !IsErrNotFound(err) {
					util.WarningOnce("Unable to remove network %s: %v", netName, err)
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
