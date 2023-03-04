package netutil

import (
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/util"
	"net"
	"os"
	"syscall"
)

// IsPortActive checks to see if the given port on docker IP is answering.
func IsPortActive(port string) bool {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		util.Warning("Failed to get docker IP address: %v", err)
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
