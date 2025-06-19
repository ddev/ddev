package hostname

import (
	"fmt"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/goodhosts/hostsfile"
	"os"
	exec2 "os/exec"
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
	ddevhostnameBinary := getDdevHostnameBInary()
	out, err := escalateHostsManipulation([]string{ddevhostnameBinary, "hostname", hostname, ip})
	return out, err
}

// EscalateToRemoveHostEntry runs the required ddev_hostname command to remove the entry,
// does it with sudo on the correct platform.
func EscalateToRemoveHostEntry(hostname string, ip string) (string, error) {
	ddevhostnameBinary := getDdevHostnameBInary()
	out, err := escalateHostsManipulation([]string{
		ddevhostnameBinary, "hostname", "--remove", hostname, ip})
	return out, err
}

func getDdevHostnameBInary() string {
	ddevBinary, _ := os.Executable()
	ddevDir := filepath.Dir(ddevBinary)
	ddevhostnameBinary := filepath.Join(ddevDir, ddevhostnameBinary)
	if runtime.GOOS == "windows" || nodeps.IsWSL2() {
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

// IsDdevHostnameAvailable checks to see if we can use ddev.exe on Windows side
func IsDdevHostnameAvailable() bool {
	ddevHostnameBinary := getDdevHostnameBInary()
	if !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt && !ddevHostnameAvailable && nodeps.IsWSL2() {
		_, err := exec2.LookPath(ddevHostnameBinary)
		if err != nil {
			util.Warning("%s not found, please install it; err=%v", ddevhostnameBinary, err)
			ddevHostnameAvailable = false
			return ddevHostnameAvailable
		}
		out, err := exec.RunHostCommand("ddev_hostname.exe", "--version")
		if err != nil {
			util.Warning("Unable to run ddev_hostname.exe, please check it on Windows side; err=%v; output=%s", err, out)
			ddevHostnameAvailable = false
			return ddevHostnameAvailable
		}

		_, err = exec2.LookPath("gsudo.exe")
		if err != nil {
			util.Warning("gsudo.exe not found in $PATH, please install DDEV on Windows side; err=%v", err)
			ddevHostnameAvailable = false
			return ddevHostnameAvailable
		}
		ddevHostnameAvailable = true
	}
	return ddevHostnameAvailable
}
