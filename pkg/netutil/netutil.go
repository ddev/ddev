package netutil

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// IsPortActive checks to see if the given port on Docker IP is answering.
func IsPortActive(port string) bool {
	dialTimeout := 0 * time.Millisecond
	if nodeps.IsWSL2() {
		dialTimeout = 200 * time.Millisecond
	}

	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		util.Warning("Failed to get Docker IP address: %v", err)
		return false
	}

	// Skip port check for remote Docker hosts (non-local IPs)
	// Remote IPs may cause timeouts and false positives
	if parsedIP := net.ParseIP(dockerIP); parsedIP != nil && !parsedIP.IsLoopback() {
		localIPs, _ := GetLocalIPs()
		if !slices.Contains(localIPs, dockerIP) {
			util.Verbose("Skipping port check for remote Docker host %s:%s", dockerIP, port)
			return false
		}
	}

	util.Verbose("Checking if port %s is active", port)
	conn, err := net.DialTimeout("tcp", dockerIP+":"+port, dialTimeout)

	// If we were able to connect, something is listening on the port.
	if err == nil {
		_ = conn.Close()
		return true
	}

	// In WSL2 mirrored mode, when we test an unused port, we just get a timeout
	// Assume that the port is available (not active) in that situation.
	// This seems to be caused by https://github.com/microsoft/WSL/issues/10855
	// We don't have a way to know whether WSL2 in mirrored mode, but
	// we use the longer timeout in WSL2 and assume that timeout is unoccupied.
	if nodeps.IsWSL2() {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			util.Debug("In WSL2 and port %s is probably not active; timeout", port)
			return false
		}
	}

	// If we get ECONNREFUSED the port is not active.
	oe, ok := err.(*net.OpError)
	if ok {
		syscallErr, ok := oe.Err.(*os.SyscallError)

		// On Windows, WSAECONNREFUSED (10061) results instead of ECONNREFUSED. And golang doesn't seem to have it.
		var WSAECONNREFUSED syscall.Errno = 10061

		if ok && (syscallErr.Err == syscall.ECONNREFUSED || syscallErr.Err == WSAECONNREFUSED) {
			util.Verbose("port %s shows connection refused so not active", port)
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

// NormalizeURL removes the port from a URL if it is the default port for the scheme
func NormalizeURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		util.Warning("Failed to parse URL %s: %v", rawURL, err)
		return ""
	}

	if (parsedURL.Scheme == "http" && parsedURL.Port() == "80") ||
		(parsedURL.Scheme == "https" && parsedURL.Port() == "443") {
		parsedURL.Host = strings.TrimSuffix(parsedURL.Host, ":"+parsedURL.Port())
	}

	return parsedURL.String()
}
