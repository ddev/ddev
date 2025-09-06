package dockerutil

import (
	"strings"

	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/docker/api/types/network"
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

	// Gitpod has Docker 20.10+ so the docker-compose has already gotten the host-gateway
	// However, NFS will never be used on Gitpod.
	case nodeps.IsGitpod():
		break
	case nodeps.IsCodespaces():
		break

	case nodeps.IsWSL2() && IsDockerDesktop():
		// If IDE is on Windows, return; we don't have to do anything.
		break

	case nodeps.IsWSL2() && !IsDockerDesktop():

		nfsAddr = wsl2GetWindowsHostIP()

	// Docker on Linux doesn't define host.docker.internal
	// so we need to go get the bridge IP address
	// Docker Desktop defines host.docker.internal itself.
	case nodeps.IsLinux():
		// Look up info from the bridge network
		// We can't use the Docker host because that's for inside the container,
		// and this is for setting up the network interface
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
				nfsAddr = n.IPAM.Config[0].Gateway
			} else {
				util.Warning("Unable to determine Docker bridge gateway - no gateway")
			}
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
