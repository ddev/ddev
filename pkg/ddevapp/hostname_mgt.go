package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevhosts"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	goodhosts "github.com/goodhosts/hostsfile"
	"net"
	"os"
	exec2 "os/exec"
	"runtime"
	"strings"
)

// windowsDdevExeAvailable says if ddev.exe is available on Windows side
var windowsDdevExeAvailable bool

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
			util.Warning("unable to run ddev.exe, please check it on Windows side; err=%v; output=%s", err, out)
			windowsDdevExeAvailable = false
			return windowsDdevExeAvailable
		}

		_, err = exec2.LookPath("sudo.exe")
		if err != nil {
			util.Warning("sudo.exe not found in $PATH, please install DDEV on Windows side; err=%v", err)
			windowsDdevExeAvailable = false
			return windowsDdevExeAvailable
		}
		windowsDdevExeAvailable = true
	}
	return windowsDdevExeAvailable
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
		return false, fmt.Errorf("Unable to open hosts file: %v", err)
	}
	return hosts.Has(dockerIP, hostname), nil
}

// AddHostsEntriesIfNeeded will run sudo ddev hostname to the site URL to the host's /etc/hosts.
// This should be run without admin privs; the ddev hostname command will handle escalation.
func (app *DdevApp) AddHostsEntriesIfNeeded() error {
	var err error
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	if os.Getenv("DDEV_NONINTERACTIVE") == "true" {
		util.Warning("Not trying to add hostnames because DDEV_NONINTERACTIVE=true")
		return nil
	}

	for _, name := range app.GetHostnames() {

		// If we're able to resolve the hostname via DNS or otherwise we
		// don't have to worry about this. This will allow resolution
		// of *.ddev.site for example
		if app.UseDNSWhenPossible && globalconfig.IsInternetActive() {
			// If they have provided "*.<name>" then look up the suffix
			checkName := strings.TrimPrefix(name, "*.")
			hostIPs, err := net.LookupHost(checkName)

			// If we had successful lookup and dockerIP matches
			// with adding to hosts file.
			if err == nil && len(hostIPs) > 0 && hostIPs[0] == dockerIP {
				continue
			}
		}

		// We likely won't hit the hosts.Has() as true because
		// we already did a lookup. But check anyway.
		exists, err := IsHostnameInHostsFile(name)
		if exists {
			continue
		}
		if err != nil {
			util.Warning("unable to open hosts file: %v", err)
			continue
		}
		util.Warning("The hostname %s is not currently resolvable, trying to add it to the hosts file", name)

		out, err := escalateToAddHostEntry(name, dockerIP)
		if err != nil {
			return err
		}
		util.Success(out)
	}

	return nil
}

// AddHostEntry adds an entry to default hosts file
// This is only used by `ddev hostname` and only used with admin privs
func AddHostEntry(name string, ip string) error {
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		util.Warning("You must manually add the following entry to your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname %s %s'", ip, name, name, ip)
		return nil
	}

	hosts, err := goodhosts.NewHosts()
	if err != nil {
		return err
	}
	err = hosts.Add(ip, name)
	if err != nil {
		return err
	}
	err = hosts.Flush()
	return err
}

// RemoveHostsEntriesIfNeeded will remove the site URL from the host's /etc/hosts.
// This should be run without administrative privileges and will escalate
// where needed.
func (app *DdevApp) RemoveHostsEntriesIfNeeded() error {
	if os.Getenv("DDEV_NONINTERACTIVE") == "true" {
		util.Warning("Not trying to remove hostnames because DDEV_NONINTERACTIVE=true")
		return nil
	}

	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	for _, name := range app.GetHostnames() {
		exists, err := IsHostnameInHostsFile(name)
		if !exists {
			continue
		}
		if err != nil {
			util.Warning("unable to open hosts file: %v", err)
			continue
		}

		_, err = escalateToRemoveHostEntry(name, dockerIP)

		if err != nil {
			util.Warning("Failed to remove host entry %s: %v", name, err)
		}
	}

	return nil
}

// RemoveHostEntry removes named /etc/hosts entry if it exists
// This should be run with administrative privileges only and used by
// ddev hostname only
func RemoveHostEntry(name string, ip string) error {
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		util.Warning("You must manually add the following entry to your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname %s %s'", ip, name, name, ip)
		return nil
	}

	hosts, err := goodhosts.NewHosts()
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

// escalateToAddHostEntry runs the required ddev hostname command to add the entry,
// does it with sudo on the correct platform.
func escalateToAddHostEntry(hostname string, ip string) (string, error) {
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

// escalateToRemoveHostEntry runs the required ddev hostname command to remove the entry,
// does it with sudo on the correct platform.
func escalateToRemoveHostEntry(hostname string, ip string) (string, error) {
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
		return "", fmt.Errorf("could not get home directory for current user. is it set?")
	}

	if (nodeps.IsWSL2() && !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt) && !IsWindowsDdevExeAvailable() {
		return "", fmt.Errorf("ddev.exe is not installed on the Windows side, please install it with 'choco install -y ddev'. It is used to manage the Windows hosts file")
	}
	c := []string{"sudo", "--preserve-env=HOME"}
	if (runtime.GOOS == "windows" || nodeps.IsWSL2()) && !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt {
		c = []string{"sudo.exe"}
	}
	c = append(c, args...)
	output.UserOut.Printf("ddev needs to run with administrative privileges.\nYou may be required to enter your password for sudo or allow escalation. ddev is about to issue the command:\n  %s\n", strings.Join(c, ` `))

	out, err = exec.RunHostCommand(c[0], c[1:]...)
	return out, err
}
