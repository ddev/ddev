package hostname

import (
	"fmt"
	"os"
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
	ddevHostnameBinary := GetDdevHostnameBinary()
	out, err := elevateHostsManipulation([]string{ddevHostnameBinary, hostname, ip})
	return out, err
}

// ElevateToRemoveHostEntry runs the required ddev-hostname command to remove the entry,
func ElevateToRemoveHostEntry(hostname string, ip string) (string, error) {
	ddevHostnameBinary := GetDdevHostnameBinary()
	out, err := elevateHostsManipulation([]string{ddevHostnameBinary, "--remove", hostname, ip})
	return out, err
}

// GetDdevHostnameBinary returns the path to the ddev-hostname or ddev-hostname.exe binary
// It must exist in the PATH
func GetDdevHostnameBinary() string {
	ddevHostnameBinary := ddevHostnameBinary
	if runtime.GOOS == "windows" || (nodeps.IsWSL2() && !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt) {
		ddevHostnameBinary = ddevHostnameWindowsBinary
	}
	util.Debug("ddevHostnameBinary=%s", ddevHostnameBinary)
	return ddevHostnameBinary
}

// elevateHostsManipulation uses escalation (sudo or runas) to manipulate the hosts file.
func elevateHostsManipulation(args []string) (out string, err error) {
	// We can't escalate in tests, and they know how to deal with it.
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		util.Warning("DDEV_NONINTERACTIVE is set. You must manually run '%s'", strings.Join(args, " "))
		return "", nil
	}

	if !isDdevHostnameAvailable() {
		return "Binary not found", fmt.Errorf("%s is not installed, please install it, see https://ddev.readthedocs.io/en/stable/users/usage/commands/#hostname", ddevHostnameBinary)
	}

	c := args
	output.UserOut.Printf("%s needs to run with administrative privileges.\nThis is required to add unresolvable hostnames to the hosts file.\nYou may need to enter your password for sudo or allow escalation.", GetDdevHostnameBinary())
	output.UserOut.Printf("DDEV will issue the command:\n  %s\n", strings.Join(c, ` `))

	out, err = exec.RunHostCommand(c[0], c[1:]...)
	return strings.TrimSpace(out), err
}

// ddevHostnameAvailable says if ddev-hostname/ddev-hostname.exe is available
var ddevHostnameAvailable bool

// isDdevHostnameAvailable checks to see if we can use ddev-hostname
func isDdevHostnameAvailable() bool {
	ddevHostnameBinary := GetDdevHostnameBinary()
	// Use ddev-hostname --version to check if ddev-hostname is available
	out, err := exec.RunHostCommand(ddevHostnameBinary, "--version")
	if err == nil {
		ddevHostnameAvailable = true
	} else {
		util.Warning("Unable to run %s, please check it; err=%v; output=%s", ddevHostnameBinary, err, strings.TrimSpace(out))
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
