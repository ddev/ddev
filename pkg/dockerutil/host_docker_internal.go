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
	"github.com/ddev/ddev/pkg/util"
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
			message = fmt.Sprintf("host.docker.internal='%s' because xdebug_ide_location=%s, see https://docs.ddev.com/en/stable/users/configuration/config/#xdebug_ide_location", ipAddress, globalconfig.DdevGlobalConfig.XdebugIDELocation)

		case globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationContainer:
			// If the IDE is actually listening inside container, then localhost/127.0.0.1 should work.
			ipAddress = "127.0.0.1"
			message = fmt.Sprintf("host.docker.internal='%s' because xdebug_ide_location=%s, see https://docs.ddev.com/en/stable/users/configuration/config/#xdebug_ide_location", ipAddress, globalconfig.DdevGlobalConfig.XdebugIDELocation)

		case IsColima():
			// Lima specifies this as a named explicit IP address at this time
			// see https://lima-vm.io/docs/config/network/user/#host-ip-19216852
			ipAddress = "192.168.5.2"
			message = fmt.Sprintf("host.docker.internal='%s' because IsColima", ipAddress)

		case nodeps.IsGitpod():
			message = fmt.Sprintf("host.docker.internal='%s' because IsGitpod uses 'host-gateway' in extra_hosts", ipAddress)

		case nodeps.IsCodespaces():
			message = fmt.Sprintf("host.docker.internal='%s' because IsCodespaces uses 'host-gateway' in extra_hosts", ipAddress)

		case nodeps.IsWSL2() && IsDockerDesktop():
			// If IDE is on Windows, return; we don't have to do anything.
			message = fmt.Sprintf("host.docker.internal='%s' because IsWSL2 and IsDockerDesktop", ipAddress)

		case nodeps.IsWSL2() && globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationWSL2:
			// If IDE is inside WSL2 then the normal Linux processing should work
			message = fmt.Sprintf("host.docker.internal='%s' because xdebug_ide_location=%s uses 'host-gateway' in extra_hosts, see https://docs.ddev.com/en/stable/users/configuration/config/#xdebug_ide_location", ipAddress, globalconfig.DdevGlobalConfig.XdebugIDELocation)

		case nodeps.IsWSL2() && !nodeps.IsWSL2MirroredMode() && !IsDockerDesktop():
			// Microsoft instructions for finding Windows IP address at
			// https://learn.microsoft.com/en-us/windows/wsl/networking#accessing-windows-networking-apps-from-linux-host-ip
			// If IDE is on Windows, we have to parse /etc/resolv.conf
			ipAddress = wsl2GetWindowsHostIP()
			message = fmt.Sprintf("host.docker.internal='%s' because IsWSL2 and !IsDockerDesktop; received from 'ip -4 route show default'", ipAddress)

		case nodeps.IsWSL2MirroredMode() && !IsDockerDesktop():
			if windowsReachableIP, err := getWindowsReachableIP(); err == nil && windowsReachableIP != "" {
				ipAddress = windowsReachableIP
				message = fmt.Sprintf("host.docker.internal='%s' because IsWSL2MirroredMode and !IsDockerDesktop; received from getWindowsReachableIP()", ipAddress)
			} else {
				message = fmt.Sprintf("host.docker.internal='%s' because IsWSL2MirroredMode and !IsDockerDesktop; getWindowsReachableIP() failed", ipAddress)
			}

		case nodeps.IsLinux():
			// host.docker.internal is already taken care of by extra_hosts in docker-compose
			message = fmt.Sprintf("host.docker.internal='%s' because IsLinux uses 'host-gateway' in extra_hosts", ipAddress)

		default:
			message = fmt.Sprintf("host.docker.internal='%s' because no other case was discovered", ipAddress)
		}

		if ipAddress != "" {
			extraHost = ipAddress
		} else if (nodeps.IsLinux() && !nodeps.IsWSL2() && !IsColima()) ||
			(nodeps.IsWSL2() && globalconfig.DdevGlobalConfig.XdebugIDELocation == globalconfig.XdebugIDELocationWSL2) {
			// Use "host-gateway" for Docker on Linux and for WSL2 with IDE in WSL2
			extraHost = "host-gateway"
		}
		hostDockerInternal = &HostDockerInternal{
			IPAddress: ipAddress,
			ExtraHost: extraHost,
			Message:   message,
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

// wsl2GetWindowsHostIP uses 'ip -4 route show default' to get the Windows IP address
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
