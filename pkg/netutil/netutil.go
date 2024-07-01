package netutil

import (
	"net"
	"os"
	"slices"
	"syscall"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/util"
)

// IsPortActive checks to see if the given port on Docker IP is answering.
func IsPortActive(port string) bool {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		util.Warning("Failed to get Docker IP address: %v", err)
		return false
	}

	conn, err := net.Dial("tcp", dockerIP+":"+port)

	// If we were able to connect, something is listening on the port.
	if err == nil {
		_ = conn.Close()
		return true
	}
	// If we get ECONNREFUSED the port is not active.
	oe, ok := err.(*net.OpError)
	if ok {
		syscallErr, ok := oe.Err.(*os.SyscallError)

		// On Windows, WSAECONNREFUSED (10061) results instead of ECONNREFUSED. And golang doesn't seem to have it.
		var WSAECONNREFUSED syscall.Errno = 10061

		if ok && (syscallErr.Err == syscall.ECONNREFUSED || syscallErr.Err == WSAECONNREFUSED) {
			return false
		}
	}
	// Otherwise, hmm, something else happened. It's not a fatal or anything.
	util.Warning("Unable to properly check port status: %v", oe)
	return false
}

// IsLocalIP returns true if the provided IP address is
// assigned to the local computer
func IsLocalIP(ipAddress string) bool {
	dockerIP, err := dockerutil.GetDockerIP()

	if err != nil {
		util.Warning("Failed to get Docker IP address: %v", err)
		return false
	}

	if ipAddress == dockerIP {
		return true
	}

	localIPs, err := GetLocalIPs()

	if err != nil {
		util.Warning("Failed to get local IPs: %v", err)
		return false
	}

	// Check if the parsed IP address is local using slices.Contains
	return slices.Contains(localIPs, ipAddress)
}

// GetLocalIPs() returns IP addresses associated with the machine
func GetLocalIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var localIPs []string
	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			if v.IP.IsLoopback() {
				continue
			}
			localIPs = append(localIPs, v.IP.String())
		case *net.IPAddr:
			if v.IP.IsLoopback() {
				continue
			}
			localIPs = append(localIPs, v.IP.String())
		}
	}

	return localIPs, nil
}
