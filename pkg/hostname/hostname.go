package hostname

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevhosts"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/goodhosts/hostsfile"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const ddevhostnameBinary = "ddev_hostname"
const ddevhostnameWindowsBinary = ddevhostnameBinary + ".exe"

// AddHostEntry adds an entry to default hosts file
// This is only used by `ddev hostname` and only used with admin privs
func AddHostEntry(name string, ip string) error {
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		util.Warning("You must manually add the following entry to your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname %s %s'", ip, name, name, ip)
		return nil
	}

	osHostsFilePath := os.ExpandEnv(filepath.FromSlash(hostsfile.HostsFilePath))

	hosts, err := hostsfile.NewCustomHosts(osHostsFilePath)

	if err != nil {
		return err
	}
	err = hosts.Add(ip, name)
	if err != nil {
		return err
	}
	hosts.HostsPerLine(8)
	err = hosts.Flush()
	return err
}

// RemoveHostEntry removes named /etc/hosts entry if it exists
// This should be run with administrative privileges only and used by
// DDEV hostname only
func RemoveHostEntry(name string, ip string) error {
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		util.Warning("You must manually add the following entry to your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname %s %s'", ip, name, name, ip)
		return nil
	}

	hosts, err := hostsfile.NewHosts()
	if err != nil {
		return err
	}
	err = hosts.Remove(ip, name)
	if err != nil {
		return err
	}
	err = hosts.Flush()
	return err
}

// EscalateToAddHostEntry runs the required DDEV hostname command to add the entry,
// does it with sudo on the correct platform.
func EscalateToAddHostEntry(hostname string, ip string) (string, error) {
	ddevhostnameBinary := getDdevHostnameBinary()
	out, err := escalateHostsManipulation([]string{ddevhostnameBinary, "hostname", hostname, ip})
	return out, err
}

// EscalateToRemoveHostEntry runs the required ddev_hostname command to remove the entry,
// does it with sudo on the correct platform.
func EscalateToRemoveHostEntry(hostname string, ip string) (string, error) {
	ddevhostnameBinary := getDdevHostnameBinary()
	out, err := escalateHostsManipulation([]string{
		ddevhostnameBinary, "hostname", "--remove", hostname, ip})
	return out, err
}

// getDdevHostnameBinary returns the path to the ddev_hostname or ddev_hostname.exe binary
func getDdevHostnameBinary() string {
	ddevBinary, _ := os.Executable()
	ddevDir := filepath.Dir(ddevBinary)
	ddevhostnameBinary := filepath.Join(ddevDir, ddevhostnameBinary)
	if runtime.GOOS == "windows" || (nodeps.IsWSL2() && !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt) {
		ddevhostnameBinary = filepath.Join(ddevDir, ddevhostnameWindowsBinary)
	}
	return ddevhostnameBinary
}

// escalateHostsManipulation uses escalation (sudo or runas) to manipulate the hosts file.
func escalateHostsManipulation(args []string) (out string, err error) {
	// We can't escalate in tests, and they know how to deal with it.
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		util.Warning("DDEV_NONINTERACTIVE is set. You must manually run '%s'", strings.Join(args, " "))
		return "", nil
	}
	_, err = os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory for current user. Is it set?")
	}

	if !IsDdevHostnameAvailable() {
		return "", fmt.Errorf("%s is not installed, please install it.", ddevhostnameBinary)
	}
	c := []string{"sudo", "--preserve-env=HOME"}
	if (runtime.GOOS == "windows" || nodeps.IsWSL2()) && !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt {
		c = []string{"gsudo.exe"}
	}
	c = append(c, args...)
	output.UserOut.Printf("DDEV needs to run with administrative privileges.\nThis is required to add unresolvable hostnames to the hosts file.\nYou may need to enter your password for sudo or allow escalation.\nDDEV is about to issue the command:\n  %s\n", strings.Join(c, ` `))

	out, err = exec.RunHostCommand(c[0], c[1:]...)
	return out, err
}

// ddevHostnameAvailable says if ddev_hostname/ddev_hostname.exe is available
var ddevHostnameAvailable bool

// IsDdevHostnameAvailable checks to see if we can use ddev_hostname
func IsDdevHostnameAvailable() bool {
	ddevHostnameBinary := getDdevHostnameBinary()
	// Use ddev_hostname --version to check if ddev_hostname is available
	out, err := exec.RunHostCommand(ddevHostnameBinary, "--version")
	if err == nil {
		ddevHostnameAvailable = true
	} else {
		util.Warning("Unable to run %s, please check it; err=%v; output=%s", ddevhostnameBinary, err, out)
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

	var hosts = &ddevhosts.DdevHosts{}
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
