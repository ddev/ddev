package dockerutil

import (
	"strings"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// GetNFSServerAddr gets the address that can be used for the NFS server.
// It's almost the same as GetDockerHostInternalIP() but we have
// to get the actual addr in the case of Linux; still, Linux rarely
// is used with NFS. Returns "host.docker.internal" by default (not empty)
func GetNFSServerAddr() (string, error) {
	nfsAddr := "host.docker.internal"

	switch {
	case IsColima():
		// Lima specifies this as a named explicit IP address at this time
		// see https://lima-vm.io/docs/config/network/user/#host-ip-19216852
		nfsAddr = "192.168.5.2"

	case nodeps.IsCodespaces():
		break

	case nodeps.IsWSL2() && IsDockerDesktop():
		// If IDE is on Windows, return; we don't have to do anything.
		break

	case nodeps.IsWSL2() && !IsDockerDesktop():

		wsl2WindowsHostIP, err := getWSL2WindowsHostIP()
		if err != nil {
			util.Warning("Unable to determine WSL2 Windows host IP address: %v", err)
		} else {
			nfsAddr = wsl2WindowsHostIP
		}

	// Docker on Linux doesn't define host.docker.internal
	// so we need to go get the bridge IP address
	// Docker Desktop defines host.docker.internal itself.
	case nodeps.IsLinux():
		// Look up info from the bridge network
		// We can't use the Docker host because that's for inside the container,
		// and this is for setting up the network interface
		dockerBridgeIP, err := getDockerLinuxBridgeIP()
		if err != nil {
			util.Warning("Unable to determine Docker bridge gateway IP address: %v", err)
		} else {
			nfsAddr = dockerBridgeIP
		}
	}

	return nfsAddr, nil
}

// MassageWindowsNFSMount changes C:\Path\to\something to /c/Path/to/something
func MassageWindowsNFSMount(mountPoint string) string {
	if string(mountPoint[1]) == ":" {
		pathPortion := strings.ReplaceAll(mountPoint[2:], `\`, "/")
		drive := string(mountPoint[0])
		// Because we use $HOME to get home in exports, and $HOME has /c/Users/xxx
		// change the drive to lower case.
		mountPoint = "/" + strings.ToLower(drive) + pathPortion
	}
	return mountPoint
}
