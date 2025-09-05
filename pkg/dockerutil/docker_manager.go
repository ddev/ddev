package dockerutil

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	ddevexec "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/cli/cli/version"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
)

// dockerManager manages Docker client configuration and connection state
// Some of these values are set on demand when first requested
type dockerManager struct {
	goContext                   context.Context    // Go context for Docker API calls
	client                      client.APIClient   // Docker API for making calls to Docker daemon
	cli                         *command.DockerCli // Docker CLI for getting dockerContextName and host
	dockerContextName           string             // Current Docker context name (e.g., "default", "desktop-linux")
	host                        string             // Docker daemon URL (e.g., "unix:///var/run/docker.sock")
	hostSanitized               string             // Docker host with special characters removed
	hostSanitizedErr            error              // Error from Docker host sanitization, if any
	hostIP                      string             // IP address of Docker host
	hostIPErr                   error              // Error from Docker host IP lookup, if any
	info                        system.Info        // Docker system information from daemon (version, OS, etc.)
	hostDockerInternalDetected  bool               // Whether host.docker.internal detection has been attempted
	hostDockerInternalIP        string             // Host IP for host.docker.internal, can be empty
	hostDockerInternalExtraHost string             // Value for Docker extra_hosts config ("host-gateway" or IP, or empty)
	serverVersion               types.Version      // Docker server version information
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
		opts := flags.NewClientOptions()
		sDockerManagerErr = sDockerManager.cli.Initialize(opts)
		if sDockerManagerErr != nil {
			return
		}
		sDockerManager.dockerContextName = sDockerManager.cli.CurrentContext()
		sDockerManager.host = sDockerManager.cli.DockerEndpoint().Host
		util.Verbose("getDockerManagerInstance(): dockerContextName=%s, host=%s", sDockerManager.dockerContextName, sDockerManager.host)
		sDockerManager.goContext = context.Background()
		// Set the Docker CLI version for User-Agent header
		version.Version = "ddev-" + versionconstants.DdevVersion
		// We can't use sDockerManager.cli.Client(), see https://github.com/docker/cli/issues/4489
		// That's why we create a new client from flags to catch errors
		sDockerManager.client, sDockerManagerErr = command.NewAPIClientFromFlags(
			opts,
			sDockerManager.cli.ConfigFile(),
		)
		if sDockerManagerErr != nil {
			return
		}
		sDockerManager.serverVersion, sDockerManagerErr = sDockerManager.client.ServerVersion(sDockerManager.goContext)
		if sDockerManagerErr != nil {
			return
		}
		sDockerManager.info, sDockerManagerErr = sDockerManager.client.Info(sDockerManager.goContext)
		if sDockerManagerErr != nil {
			return
		}
	})
	return sDockerManager, sDockerManagerErr
}

// GetDockerClient returns the Go context and the Docker API client
func GetDockerClient() (context.Context, client.APIClient, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return nil, nil, err
	}
	return dm.goContext, dm.client, nil
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

// GetDockerHostSanitized returns Docker host but with all special characters removed
// It stands in for Docker context name, but Docker context name is not a reliable indicator
func GetDockerHostSanitized() (string, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return "", err
	}
	if dm.hostSanitized != "" || dm.hostSanitizedErr != nil {
		return dm.hostSanitized, dm.hostSanitizedErr
	}
	// Make it shorter so we don't hit Mutagen 63-char limit
	dockerHost := strings.TrimPrefix(dm.host, "unix://")
	dockerHost = strings.TrimSuffix(dockerHost, "docker.sock")
	dockerHost = strings.Trim(dockerHost, "/.")
	// Convert remaining descriptor to alphanumeric
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		dm.hostSanitizedErr = err
		return "", dm.hostSanitizedErr
	}
	alphaOnly := reg.ReplaceAllString(dockerHost, "-")
	dm.hostSanitized = alphaOnly
	return dm.hostSanitized, dm.hostSanitizedErr
}

// GetDockerIP returns either the default Docker IP address (127.0.0.1)
// or the value as configured by Docker host (if it is a tcp:// URL)
func GetDockerIP() (string, error) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return "", err
	}
	if dm.hostIP != "" || dm.hostIPErr != nil {
		return dm.hostIP, dm.hostIPErr
	}
	dockerHostURL, err := url.Parse(dm.host)
	if err != nil {
		dm.hostIPErr = fmt.Errorf("failed to parse dm.host=%s: %v", dm.host, err)
		return "", dm.hostIPErr
	}
	hostPart := dockerHostURL.Hostname()
	if hostPart == "" {
		dm.hostIP = "127.0.0.1"
		return dm.hostIP, dm.hostIPErr
	}
	// Check to see if the hostname we found is an IP address
	addr := net.ParseIP(hostPart)
	if addr == nil {
		// If it wasn't an IP address, look it up to get IP address
		ip, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", hostPart)
		if err == nil && len(ip) > 0 {
			hostPart = ip[0].String()
		} else {
			dm.hostIPErr = fmt.Errorf("failed to look up IP address for dm.host=%s, hostname=%s: %v", dm.host, hostPart, err)
			return "", dm.hostIPErr
		}
	}
	dm.hostIP = hostPart
	return dm.hostIP, dm.hostIPErr
}

// ResetDockerIPForDockerHost resets the cached Docker IP address for the given Docker host.
// Used for testing only.
func ResetDockerIPForDockerHost(host string) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return
	}
	dm.host = host
	dm.hostIP = ""
	dm.hostIPErr = nil
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

// GetHostDockerInternal determines the correct host.docker.internal configuration for containers.
// Returns two values: (hostDockerInternalIP, hostDockerInternalExtraHost)
// - hostDockerInternalIP: The IP address that containers should use to reach the host, or empty string if not needed
// - hostDockerInternalExtraHost: The value to use in Docker's extra_hosts for host.docker.internal
//
// The function handles platform-specific cases:
// - Docker Desktop: Uses built-in host.docker.internal (returns "", "")
// - Linux: Uses host-gateway (returns "", "host-gateway")
// - WSL2 scenarios: Detects Windows host IP via routing table (returns "x.x.x.x", "x.x.x.x")
// - Colima: Uses fixed IP 192.168.5.2 (returns "192.168.5.2", "192.168.5.2")
// - IDE location overrides: Respects global Xdebug IDE location settings
func GetHostDockerInternal() (string, string) {
	dm, err := getDockerManagerInstance()
	if err != nil {
		return "", ""
	}

	if dm.hostDockerInternalDetected {
		return dm.hostDockerInternalIP, dm.hostDockerInternalExtraHost
	}

	dm.hostDockerInternalDetected = true
	dm.hostDockerInternalIP = ""
	dm.hostDockerInternalExtraHost = ""

	switch {
	case nodeps.IsIPAddress(globalconfig.DdevGlobalConfig.XdebugIDELocation):
		// If the IDE is actually listening inside container, then localhost/127.0.0.1 should work.
		dm.hostDockerInternalIP = globalconfig.DdevGlobalConfig.XdebugIDELocation
		util.Debug("host.docker.internal='%s' derived from globalconfig.DdevGlobalConfig.XdebugIDELocation", dm.hostDockerInternalIP)

	case globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationContainer:
		// If the IDE is actually listening inside container, then localhost/127.0.0.1 should work.
		dm.hostDockerInternalIP = "127.0.0.1"
		util.Debug("host.docker.internal='%s' because globalconfig.DdevGlobalConfig.XdebugIDELocation=%s", dm.hostDockerInternalIP, globalconfig.XdebugIDELocationContainer)

	case IsColima():
		// Lima specifies this as a named explicit IP address at this time
		// see https://lima-vm.io/docs/config/network/user/#host-ip-19216852
		dm.hostDockerInternalIP = "192.168.5.2"
		util.Debug("host.docker.internal='%s' because running on Colima", dm.hostDockerInternalIP)

	// Gitpod has Docker 20.10+ so the docker-compose has already gotten the host-gateway
	case nodeps.IsGitpod():
		util.Debug("host.docker.internal='%s' because on Gitpod", dm.hostDockerInternalIP)
		break
	case nodeps.IsCodespaces():
		util.Debug("host.docker.internal='%s' because on Codespaces", dm.hostDockerInternalIP)
		break

	case nodeps.IsWSL2() && IsDockerDesktop():
		// If IDE is on Windows, return; we don't have to do anything.
		util.Debug("host.docker.internal='%s' because IsWSL2 and IsDockerDesktop", dm.hostDockerInternalIP)
		break

	case nodeps.IsWSL2() && globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationWSL2:
		// If IDE is inside WSL2 then the normal Linux processing should work
		util.Debug("host.docker.internal='%s' because globalconfig.DdevGlobalConfig.XdebugIDELocation=%s", dm.hostDockerInternalIP, globalconfig.XdebugIDELocationWSL2)
		break

	case nodeps.IsWSL2() && !nodeps.IsWSL2MirroredMode() && !IsDockerDesktop():
		// Microsoft instructions for finding Windows IP address at
		// https://learn.microsoft.com/en-us/windows/wsl/networking#accessing-windows-networking-apps-from-linux-host-ip
		// If IDE is on Windows, we have to parse /etc/resolv.conf
		dm.hostDockerInternalIP = wsl2GetWindowsHostIP()
		util.Debug("host.docker.internal='%s' because IsWSL2 and !IsDockerDesktop; received from ip -4 route show default", dm.hostDockerInternalIP)
		break

	case nodeps.IsWSL2MirroredMode() && !IsDockerDesktop():
		if ip, err := getWindowsReachableIP(); err == nil && ip != "" {
			dm.hostDockerInternalIP = ip
			util.Debug("host.docker.internal='%s' because IsWSL2MirroredMode and !IsDockerDesktop; received from getWindowsReachableIP()", dm.hostDockerInternalIP)
		}
		break

	// Docker on Linux doesn't define host.docker.internal
	// so we need to go get the bridge IP address.
	case nodeps.IsLinux():
		// host.docker.internal is already taken care of by extra_hosts in docker-compose
		// see condition for dm.hostDockerInternalExtraHost below
		util.Debug("host.docker.internal='%s' because IsLinux uses 'host-gateway' in extra_hosts", dm.hostDockerInternalIP)
		break

	default:
		util.Debug("host.docker.internal='%s' because no other case was discovered", dm.hostDockerInternalIP)
		break
	}

	if dm.hostDockerInternalIP != "" {
		dm.hostDockerInternalExtraHost = dm.hostDockerInternalIP
	} else if (nodeps.IsLinux() && !nodeps.IsWSL2() && !IsColima()) ||
		(nodeps.IsWSL2() && globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationWSL2) {
		// Use "host-gateway" for Docker on Linux and for WSL2 with IDE in WSL2
		dm.hostDockerInternalExtraHost = "host-gateway"
	}

	return dm.hostDockerInternalIP, dm.hostDockerInternalExtraHost
}

// getWindowsReachableIP() uses PowerShell to find a windows-side IP
// address that can be accessed from inside a container.
// This is needed for networkMode=mirrored in WSL2.
func getWindowsReachableIP() (string, error) {
	cmd := exec.Command("powershell.exe", "-Command", `
Get-NetIPAddress -AddressFamily IPv4 |
  Where-Object {
    $_.IPAddress -notlike "169.254*" -and
    $_.IPAddress -ne "127.0.0.1"
  } |
  Sort-Object InterfaceMetric |
  Select-Object -First 1 -ExpandProperty IPAddress
`)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("powershell failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// wsl2GetWindowsHostIP() uses ip -4 route show default to get the Windows IP address
// for use in determining host.docker.internal
func wsl2GetWindowsHostIP() string {
	// Get default route from WSL2
	out, err := ddevexec.RunHostCommand("ip", "-4", "route", "show", "default")

	if err != nil {
		util.Warning("Unable to run 'ip -4 route show default' to get Windows IP address")
		return ""
	}
	parts := strings.Split(out, " ")
	if len(parts) < 3 {
		util.Warning("Unable to parse output of 'ip -4 route show default', result was %v", parts)
		return ""
	}

	ip := parts[2]

	if parsedIP := net.ParseIP(ip); parsedIP == nil {
		util.Warning("Unable to validate IP address '%s' from 'ip -4 route show default'", ip)
		return ""
	}

	return ip
}
