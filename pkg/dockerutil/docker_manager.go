package dockerutil

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"

	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/cli/cli/version"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/spf13/pflag"
)

// dockerManager manages Docker client configuration and connection state
// Some of these values are set on demand when first requested
type dockerManager struct {
	goContext         context.Context            // Go context for Docker API calls
	apiClient         client.APIClient           // Docker API for making calls to Docker daemon
	cli               *command.DockerCli         // Docker CLI for getting dockerContextName and host
	dockerContextName string                     // Current Docker context name (e.g., "default", "desktop-linux")
	host              string                     // Docker daemon URL (e.g., "unix:///var/run/docker.sock")
	hostIP            string                     // IP address of Docker host
	hostIPErr         error                      // Error from Docker host IP lookup, if any
	info              system.Info                // Docker system information from daemon (version, OS, etc.)
	serverVersion     client.ServerVersionResult // Docker server version information
}

var (
	// sDockerManager is the singleton instance of dockerManager
	sDockerManager *dockerManager
	// sDockerManagerOnce ensures sDockerManager is initialized only once
	sDockerManagerOnce sync.Once
	// sDockerManagerErr is any error encountered during sDockerManager initialization
	sDockerManagerErr error
)

// getDockerManagerInstance returns the singleton instance, initializing it if needed
func getDockerManagerInstance() (*dockerManager, error) {
	sDockerManagerOnce.Do(func() {
		sDockerManager = &dockerManager{}
		// Suppress any output (stdout, stderr) from docker/cli
		sDockerManager.cli, sDockerManagerErr = command.NewDockerCli(
			command.WithCombinedStreams(io.Discard),
		)
		if sDockerManagerErr != nil {
			return
		}
		// InstallFlags and SetDefaultOptions are necessary to match
		// the plugin mode behavior to handle env vars such as
		// DOCKER_TLS and DOCKER_TLS_VERIFY.
		// See https://github.com/docker/cli/blob/master/cmd/docker-trust/trust/commands.go
		flagSet := pflag.NewFlagSet("ddev", pflag.ContinueOnError)
		opts := flags.NewClientOptions()
		opts.InstallFlags(flagSet)
		opts.SetDefaultOptions(flagSet)
		sDockerManagerErr = sDockerManager.cli.Initialize(opts)
		if sDockerManagerErr != nil {
			return
		}
		sDockerManager.dockerContextName = sDockerManager.cli.CurrentContext()
		sDockerManager.host = sDockerManager.cli.DockerEndpoint().Host
		util.Verbose("getDockerManagerInstance(): dockerContextName=%s, host=%s", sDockerManager.dockerContextName, sDockerManager.host)
		sDockerManager.hostIP, sDockerManager.hostIPErr = getDockerIPFromDockerHost(sDockerManager.host)
		sDockerManager.goContext = context.Background()
		// Set the Docker CLI version for User-Agent header
		version.Version = "ddev-" + versionconstants.DdevVersion
		// We can't use sDockerManager.cli.Client(), see https://github.com/docker/cli/issues/4489
		// That's why we create a new client from flags to catch errors
		sDockerManager.apiClient, sDockerManagerErr = command.NewAPIClientFromFlags(
			opts,
			sDockerManager.cli.ConfigFile(),
		)
		if sDockerManagerErr != nil {
			return
		}
		sDockerManager.serverVersion, sDockerManagerErr = sDockerManager.apiClient.ServerVersion(sDockerManager.goContext, client.ServerVersionOptions{})
		if sDockerManagerErr != nil {
			return
		}
		info, infoErr := sDockerManager.apiClient.Info(sDockerManager.goContext, client.InfoOptions{})
		if infoErr != nil {
			sDockerManagerErr = infoErr
			return
		}
		sDockerManager.info = info.Info
	})
	return sDockerManager, sDockerManagerErr
}

// GetDockerClient returns the Go context and the Docker API client
func GetDockerClient() (context.Context, client.APIClient, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return nil, nil, err
	}
	return dm.goContext, dm.apiClient, nil
}

// GetDockerClientInfo returns the Docker system information from the daemon
func GetDockerClientInfo() (system.Info, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return system.Info{}, err
	}
	return dm.info, nil
}

// GetDockerContextNameAndHost returns the Docker context name and host
func GetDockerContextNameAndHost() (string, string, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return "", "", err
	}
	return dm.dockerContextName, dm.host, nil
}

// GetDockerIP returns either the default Docker IP address (127.0.0.1)
// or the value as configured by Docker host (if it is a tcp:// URL)
func GetDockerIP() (string, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return "", err
	}
	return dm.hostIP, dm.hostIPErr
}

// IsRemoteDockerHost returns true if the Docker host IP is not a local address,
// indicating that Docker is running on a remote machine.
func IsRemoteDockerHost() bool {
	dockerIP, err := GetDockerIP()
	if err != nil {
		return false
	}
	parsedIP := net.ParseIP(dockerIP)
	if parsedIP == nil || parsedIP.IsLoopback() {
		return false
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return true // If we can't determine local IPs, assume remote to be safe
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.Equal(parsedIP) {
			return false
		}
	}
	return true
}

// getDockerIPFromDockerHost returns the IP address of the Docker host based on the provided Docker host string.
func getDockerIPFromDockerHost(host string) (string, error) {
	// Default to localhost
	hostIP := "127.0.0.1"
	dockerHostURL, err := url.Parse(host)
	if err != nil {
		return hostIP, fmt.Errorf("failed to parse host=%s: %v", host, err)
	}
	hostPart := dockerHostURL.Hostname()
	if hostPart == "" {
		return hostIP, nil
	}
	// Check to see if the hostname we found is an IP address
	addr := net.ParseIP(hostPart)
	if addr == nil {
		// If it wasn't an IP address, look it up to get IP address
		ip, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", hostPart)
		if err == nil && len(ip) > 0 {
			hostPart = ip[0].String()
		} else {
			return hostIP, fmt.Errorf("failed to look up IP address for host=%s, hostname=%s: %v", host, hostPart, err)
		}
	}
	hostIP = hostPart
	return hostIP, nil
}

// ResetDockerHost resets cached Docker host data in singleton (it's not thread-safe).
// Used for testing only.
func ResetDockerHost(host string) error {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return err
	}
	dm.host = host
	dm.hostIP, dm.hostIPErr = getDockerIPFromDockerHost(dm.host)
	return nil
}

// GetServerVersion gets the cached versions of Docker provider engine
// This is a struct which has all info from "docker version" command
func GetServerVersion() (client.ServerVersionResult, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return client.ServerVersionResult{}, err
	}
	return dm.serverVersion, nil
}

// GetDockerVersion gets the cached version of Docker provider engine
func GetDockerVersion() (string, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return "", err
	}
	return dm.serverVersion.Version, nil
}

// GetDockerAPIVersion gets the cached API version of Docker provider engine
// See https://docs.docker.com/engine/api/#api-version-matrix
func GetDockerAPIVersion() (string, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return "", err
	}
	return dm.serverVersion.APIVersion, nil
}
