package netutil

import (
	"fmt"
	"net"
	"net/url"
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
	// Otherwise, hmm, something else happened. It's not a fatal but can be reported.
	util.Warning("Unable to properly check port status for %s:%s: err=%v", dockerIP, port, err)
	return false
}

// HasLocalIP returns true if at least one of the provided IP addresses is
// assigned to the local computer
func HasLocalIP(ipAddresses []net.IP) bool {
	dockerIP, err := dockerutil.GetDockerIP()

	if err != nil {
		util.Warning("Failed to get Docker IP address: %v", err)
		return false
	}

	for _, ipAddress := range ipAddresses {
		if ipAddress.String() == dockerIP {
			return true
		}
	}

	localIPs, err := GetLocalIPs()

	if err != nil {
		util.Warning("Failed to get local IPs: %v", err)
		return false
	}

	// Check if the parsed IP address is local using slices.Contains
	for _, ipAddress := range ipAddresses {
		if slices.Contains(localIPs, ipAddress.String()) {
			return true
		}
	}
	return false
}

// GetLocalIPs returns IP addresses associated with the machine
func GetLocalIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var localIPs []string
	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			if v.IP.IsLoopback() || v.IP.To4() == nil {
				continue
			}
			localIPs = append(localIPs, v.IP.String())
		case *net.IPAddr:
			if v.IP.IsLoopback() || v.IP.To4() == nil {
				continue
			}
			localIPs = append(localIPs, v.IP.String())
		}
	}

	return localIPs, nil
}

// BaseURLFromFullURL returns the base url (http://hostname.example.com) from a URL, without port
func BaseURLFromFullURL(fullURL string) string {
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return ""
	}
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Hostname())
	return baseURL
}
