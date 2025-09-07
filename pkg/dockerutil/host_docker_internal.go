package dockerutil

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"

	ddevexec "github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/docker/docker/api/types/network"
)

// HostDockerInternal contains host.docker.internal configuration
type HostDockerInternal struct {
	IPAddress string // IP address that containers should use to reach the host
	ExtraHost string // Value to use in docker-compose extra_hosts for host.docker.internal
	Message   string // Debug message explaining the configuration choice
}

var (
	// hostDockerInternal is the singleton instance of HostDockerInternal
	hostDockerInternal *HostDockerInternal
	// hostDockerInternalOnce ensures hostDockerInternal is initialized only once
	hostDockerInternalOnce sync.Once
)

// GetHostDockerInternal determines the correct host.docker.internal configuration for containers.
// Returns HostDockerInternal containing IP, extra_hosts value, and debug message.
//
// The function handles platform-specific cases:
// - Docker Desktop: Uses built-in host.docker.internal
// - Linux: Uses host-gateway
// - WSL2 scenarios: Detects Windows host IP via routing table
// - Colima: Uses fixed IP 192.168.5.2
// - IDE location overrides: Respects global xdebug_ide_location settings
func GetHostDockerInternal() *HostDockerInternal {
	hostDockerInternalOnce.Do(func() {
		var ipAddress string
		var extraHost string
		var message string

		switch {
		case nodeps.IsIPAddress(globalconfig.DdevGlobalConfig.XdebugIDELocation):
			ipAddress = globalconfig.DdevGlobalConfig.XdebugIDELocation
			message = fmt.Sprintf("xdebug_ide_location=%s, see https://docs.ddev.com/en/stable/users/configuration/config/#xdebug_ide_location", globalconfig.DdevGlobalConfig.XdebugIDELocation)

		case globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationContainer:
			// If the IDE is actually listening inside container, then localhost/127.0.0.1 should work.
			ipAddress = "127.0.0.1"
			message = fmt.Sprintf("xdebug_ide_location=%s, see https://docs.ddev.com/en/stable/users/configuration/config/#xdebug_ide_location", globalconfig.DdevGlobalConfig.XdebugIDELocation)

		case IsColima():
			// Lima specifies this as a named explicit IP address at this time
			// see https://lima-vm.io/docs/config/network/user/#host-ip-19216852
			ipAddress = "192.168.5.2"
			message = "IsColima"

		case nodeps.IsGitpod():
			message = "IsGitpod uses 'host-gateway' in extra_hosts"

		case nodeps.IsCodespaces():
			message = "IsCodespaces uses 'host-gateway' in extra_hosts"

		case nodeps.IsWSL2() && IsDockerDesktop():
			// If IDE is on Windows, return; we don't have to do anything.
			message = "IsWSL2 and IsDockerDesktop"

		case nodeps.IsWSL2() && globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationWSL2:
			// If IDE is inside WSL2 then the normal Linux processing should work
			message = fmt.Sprintf("xdebug_ide_location=%s uses 'host-gateway' in extra_hosts, see https://docs.ddev.com/en/stable/users/configuration/config/#xdebug_ide_location", globalconfig.DdevGlobalConfig.XdebugIDELocation)

		case nodeps.IsWSL2() && !nodeps.IsWSL2MirroredMode() && !IsDockerDesktop():
			// Microsoft instructions for finding Windows IP address at
			// https://learn.microsoft.com/en-us/windows/wsl/networking#accessing-windows-networking-apps-from-linux-host-ip
			// If IDE is on Windows, we have to parse /etc/resolv.conf
			wsl2WindowsHostIP, err := getWSL2WindowsHostIP()
			if err != nil {
				message = fmt.Sprintf("IsWSL2 and !IsDockerDesktop; unable to get IP from getWSL2WindowsHostIP(): %v", err)
			} else {
				ipAddress = wsl2WindowsHostIP
				message = "IsWSL2 and !IsDockerDesktop; received from 'ip -4 route show default'"
			}

		case nodeps.IsWSL2MirroredMode() && !IsDockerDesktop():
			windowsReachableIP, err := getWindowsReachableIP()
			if err != nil {
				message = fmt.Sprintf("IsWSL2MirroredMode and !IsDockerDesktop; unable to get IP from getWindowsReachableIP(): %v", err)
			} else {
				ipAddress = windowsReachableIP
				message = "IsWSL2MirroredMode and !IsDockerDesktop; received from getWindowsReachableIP()"
			}

		case nodeps.IsLinux():
			// host.docker.internal is already taken care of by extra_hosts in docker-compose
			message = "IsLinux uses 'host-gateway' in extra_hosts"

		default:
			message = "no other case was discovered"
		}

		if ipAddress != "" {
			extraHost = ipAddress
		} else if (nodeps.IsLinux() && !nodeps.IsWSL2() && !IsColima()) ||
			(nodeps.IsWSL2() && globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationWSL2) {
			// Use "host-gateway" for Docker on Linux and for WSL2 with IDE in WSL2
			extraHost = "host-gateway"
			if dockerBridgeIP, err := getDockerLinuxBridgeIP(); err == nil {
				ipAddress = dockerBridgeIP
			} else {
				message = message + fmt.Sprintf("; unable to get Docker bridge IP address: %v", err)
			}
		}
		hostDockerInternal = &HostDockerInternal{
			IPAddress: ipAddress,
			ExtraHost: extraHost,
			Message:   fmt.Sprintf("host.docker.internal='%s' because %s", ipAddress, message),
		}
	})

	return hostDockerInternal
}

// getWindowsReachableIP uses PowerShell to find a windows-side IP
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

// getWSL2WindowsHostIP uses 'ip -4 route show default' to get the Windows IP address
// for use in determining host.docker.internal
func getWSL2WindowsHostIP() (string, error) {
	// Get default route from WSL2
	out, err := ddevexec.RunHostCommand("ip", "-4", "route", "show", "default")

	if err != nil {
		return "", fmt.Errorf("unable to run 'ip -4 route show default' to get Windows IP address: %v", err)
	}
	parts := strings.Split(out, " ")
	if len(parts) < 3 {
		return "", fmt.Errorf("unable to parse output of 'ip -4 route show default', result was %v", parts)
	}

	ip := parts[2]

	if parsedIP := net.ParseIP(ip); parsedIP == nil {
		return "", fmt.Errorf("unable to parse IP address '%s' from 'ip -4 route show default'", ip)
	}

	return ip, nil
}

// getDockerLinuxBridgeIP gets the IP address of the Docker bridge on Linux
func getDockerLinuxBridgeIP() (string, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return "", err
	}
	n, err := client.NetworkInspect(ctx, "bridge", network.InspectOptions{})
	if err != nil {
		return "", err
	}
	if len(n.IPAM.Config) > 0 {
		if n.IPAM.Config[0].Gateway != "" {
			bridgeIP := n.IPAM.Config[0].Gateway
			return bridgeIP, nil
		}
	}
	return "", fmt.Errorf("no gateway found in Docker bridge network")
}
