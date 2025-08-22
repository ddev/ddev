package hostname

import (
	"fmt"
	"os"
	exec2 "os/exec"
	"runtime"
	"strings"

	"github.com/ddev/ddev/pkg/ddevhosts"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

const ddevHostnameBinary = "ddev-hostname"
const ddevHostnameWindowsBinary = ddevHostnameBinary + ".exe"

// ElevateToAddHostEntry runs the required DDEV hostname command to add the entry
func ElevateToAddHostEntry(hostname string, ip string) (string, error) {
	binary := GetDdevHostnameBinary()
	out, err := elevateHostsManipulation([]string{binary, hostname, ip})
	return out, err
}

// ElevateToRemoveHostEntry runs the required ddev-hostname command to remove the entry,
func ElevateToRemoveHostEntry(hostname string, ip string) (string, error) {
	binary := GetDdevHostnameBinary()
	out, err := elevateHostsManipulation([]string{binary, "--remove", hostname, ip})
	return out, err
}

// GetDdevHostnameBinary returns the path to the ddev-hostname or ddev-hostname.exe binary
// It must exist in the PATH
func GetDdevHostnameBinary() string {
	binary := ddevHostnameBinary
	if runtime.GOOS == "windows" || (nodeps.IsWSL2() && !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt) {
		binary = ddevHostnameWindowsBinary
	}
	path, err := exec2.LookPath(binary)
	if err != nil {
		util.Debug("ddevHostnameBinary not found in PATH: %v", err)
		return binary
	}
	util.Debug("ddevHostnameBinary=%s", path)
	return path
}

// elevateHostsManipulation uses elevation (sudo or runas) to manipulate the hosts file.
func elevateHostsManipulation(args []string) (out string, err error) {
	// We can't elevate in tests, and they know how to deal with it.
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		util.Warning("DDEV_NONINTERACTIVE is set. You must manually run '%s'", strings.Join(args, " "))
		return "", nil
	}

	if !isDdevHostnameAvailable() {
		return "Binary not found", fmt.Errorf("%s is not installed, please install it, see https://docs.ddev.com/en/stable/users/usage/commands/#hostname", ddevHostnameBinary)
	}

	c := args
	output.UserOut.Printf("%s needs to run with administrative privileges.\nThis is required to add unresolvable hostnames to the hosts file.\nYou may need to enter your password for sudo or allow elevation.", GetDdevHostnameBinary())
	output.UserOut.Printf("DDEV will issue the command:\n  %s\n", strings.Join(c, ` `))

	out, err = exec.RunHostCommand(c[0], c[1:]...)
	return strings.TrimSpace(out), err
}

// ddevHostnameAvailable says if ddev-hostname/ddev-hostname.exe is available
var ddevHostnameAvailable bool

// isDdevHostnameAvailable checks to see if we can use ddev-hostname
func isDdevHostnameAvailable() bool {
	binary := GetDdevHostnameBinary()
	// Use ddev-hostname --version to check if ddev-hostname is available
	out, err := exec.RunHostCommand(binary, "--version")
	if err == nil {
		ddevHostnameAvailable = true
	} else {
		util.Warning("Unable to run %s, please check it; err=%v; output=%s", binary, err, strings.TrimSpace(out))
		ddevHostnameAvailable = false
	}
	return ddevHostnameAvailable
}

// IsHostnameInHostsFile checks to see if the hostname already exists
// On WSL2 it normally assumes that the hosts file is in WSL2WindowsHostsFile
// Otherwise it lets goodhosts decide where the hosts file is.
func IsHostnameInHostsFile(hostname string) (bool, error) {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return false, fmt.Errorf("could not get Docker IP: %v", err)
	}

	var hosts *ddevhosts.DdevHosts
	if nodeps.IsWSL2() && !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt {
		hosts, err = ddevhosts.NewCustomHosts(ddevhosts.WSL2WindowsHostsFile)
	} else {
		hosts, err = ddevhosts.New()
	}
	if err != nil {
		return false, fmt.Errorf("unable to open hosts file: %v", err)
	}
	return hosts.Has(dockerIP, hostname), nil
}
