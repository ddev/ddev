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
	ddevBinary, err := os.Executable()
	if err != nil {
		return "", err
	}
	if nodeps.IsWSL2() {
		ddevBinary = "ddev.exe"
	}
	out, err := runCommandWithSudo([]string{ddevBinary, "hostname", hostname, ip})
	return out, err
}

// EscalateToRemoveHostEntry runs the required ddev hostname command to remove the entry,
// does it with sudo on the correct platform.
func EscalateToRemoveHostEntry(hostname string, ip string) (string, error) {
	ddevBinary, err := os.Executable()
	if err != nil {
		return "", err
	}
	if nodeps.IsWSL2() {
		ddevBinary = "ddev.exe"
	}
	out, err := runCommandWithSudo([]string{ddevBinary, "hostname", "--remove", hostname, ip})
	return out, err
}

// runCommandWithSudo adds sudo to command if we aren't already running with root privs
func runCommandWithSudo(args []string) (out string, err error) {
	// We can't escalate in tests, and they know how to deal with it.
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		util.Warning("DDEV_NONINTERACTIVE is set. You must manually run '%s'", strings.Join(args, " "))
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("could not get home directory for current user. Is it set?")
	}

	if (nodeps.IsWSL2() && !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt) && !IsWindowsDdevExeAvailable() {
		return "", fmt.Errorf("ddev.exe is not installed on the Windows side, please install it with 'choco install -y ddev'. It is used to manage the Windows hosts file")
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

// TODO: Check on all platforms
// windowsDdevExeAvailable says if ddev.exe is available on Windows side
var windowsDdevExeAvailable bool

// TODO: Check on all platforms
// IsWindowsDdevExeAvailable checks to see if we can use ddev.exe on Windows side
func IsWindowsDdevExeAvailable() bool {
	if !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt && !windowsDdevExeAvailable && nodeps.IsWSL2() {
		_, err := exec2.LookPath("ddev.exe")
		if err != nil {
			util.Warning("ddev.exe not found in $PATH, please install it on Windows side; err=%v", err)
			windowsDdevExeAvailable = false
			return windowsDdevExeAvailable
		}
		out, err := exec.RunHostCommand("ddev.exe", "--version")
		if err != nil {
			util.Warning("Unable to run ddev.exe, please check it on Windows side; err=%v; output=%s", err, out)
			windowsDdevExeAvailable = false
			return windowsDdevExeAvailable
		}

		_, err = exec2.LookPath("gsudo.exe")
		if err != nil {
			util.Warning("gsudo.exe not found in $PATH, please install DDEV on Windows side; err=%v", err)
			windowsDdevExeAvailable = false
			return windowsDdevExeAvailable
		}
		windowsDdevExeAvailable = true
	}
	return windowsDdevExeAvailable
}
