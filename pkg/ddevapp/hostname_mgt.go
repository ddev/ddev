package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevhosts"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/lextoumbourou/goodhosts"
	"net"
	"os"
	exec2 "os/exec"
	"runtime"
	"strings"
)

// IsWindowsDdevExeAvailable checks to see if we can run ddev.exe on Windows side
var windowsDdevExeAvailable bool

func IsWindowsDdevExeAvailable() bool {
	if !windowsDdevExeAvailable && dockerutil.IsWSL2() {
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

// AddHostsEntriesIfNeeded will (optionally) add the site URL to the host's /etc/hosts.
func (app *DdevApp) AddHostsEntriesIfNeeded() error {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	hosts, err := ddevhosts.New()
	if err != nil {
		util.Failed("could not open hostfile: %v", err)
	}

	CheckWindowsHostsFile()

	for _, name := range app.GetHostnames() {
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
		if hosts.Has(dockerIP, name) {
			continue
		}
		util.Warning("The hostname %s is not currently resolvable, trying to add it to the hosts file", name)
		err = addHostEntry(name, dockerIP)
		if err != nil {
			return err
		}
	}

	return nil
}

// addHostEntry adds an entry to /etc/hosts
// We would have hoped to use DNS or have found the entry already in hosts
// But if it's not, try to add one.
func addHostEntry(name string, ip string) error {
	if !dockerutil.IsWSL2() || !IsWindowsDdevExeAvailable() {
		_, err := exec2.LookPath("sudo")
		if (os.Getenv("DDEV_NONINTERACTIVE") != "") || err != nil {
			util.Warning("You must manually add the following entry to your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname %s %s'", ip, name, name, ip)

			return nil
		}
	}

	ddevFullpath, err := os.Executable()
	util.CheckErr(err)

	hostnameArgs := []string{ddevFullpath, "hostname", name, ip}
	if !dockerutil.IsWSL2() || !IsWindowsDdevExeAvailable() {
		hostnameArgs = append([]string{"sudo"}, hostnameArgs...)
	}
	output.UserOut.Printf("ddev needs to add an entry to your hostfile.\nIt may require administrative privileges via the sudo command, so you may be required\nto enter your password for sudo or allow escalation. ddev is about to issue the command:\n   %s", strings.Join(hostnameArgs, " "))

	output.UserOut.Println("Please enter your password or allow escalation if prompted.")
	out, err := exec.RunHostCommand(hostnameArgs[0], hostnameArgs[1:]...)
	if err != nil {
		util.Warning("Failed to execute %s, you will need to manually execute '%s' with administrative privileges, err=%v, output=%v", strings.Join(hostnameArgs, " "), strings.Join(hostnameArgs, " "), err, out)
	}
	util.Debug("output of RunCommandPipe sudo %v=%v", strings.Join(hostnameArgs, " "), out)
	return nil
}

// RemoveHostsEntries will remote the site URL from the host's /etc/hosts.
func (app *DdevApp) RemoveHostsEntries() error {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	for _, name := range app.GetHostnames() {
		checkName := strings.TrimPrefix(name, "*.")
		hostIPs, err := net.LookupHost(checkName)

		// If we had successful lookup and dockerIP matches then continue to delete
		if err != nil || (len(hostIPs) > 0 && hostIPs[0] != dockerIP) {
			continue
		}

		if !dockerutil.IsWSL2() || !IsWindowsDdevExeAvailable() {
			hosts, err := goodhosts.NewHosts()
			if err != nil {
				util.Failed("could not open hostfile: %v", err)
			}

			if !hosts.Has(dockerIP, name) {
				continue
			}

			_, err = exec2.LookPath("sudo")
			if os.Getenv("DDEV_NONINTERACTIVE") != "" || err != nil {
				util.Warning("You must manually remove the following entry from your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname --remove %s %s", dockerIP, name, name, dockerIP)
				return nil
			}
		}

		ddevFullPath, err := os.Executable()
		util.CheckErr(err)

		hostnameArgs := []string{ddevFullPath, "hostname", "--remove", name, dockerIP}
		command := strings.Join(hostnameArgs, " ")
		util.Warning(fmt.Sprintf("    sudo %s", command))
		output.UserOut.Printf("ddev needs to remove '%s' from your hosts file.\nIt may require administrative privileges via the sudo command or escalation, so you may be required\nto enter your password for sudo. ddev is about to issue the command:\n    %s", name, strings.Join(hostnameArgs, " "))
		output.UserOut.Println("Please enter your password if prompted.")

		if !dockerutil.IsWSL2() || !IsWindowsDdevExeAvailable() {
			hostnameArgs = append([]string{"sudo"}, hostnameArgs...)
		}
		if out, err := exec.RunHostCommand(hostnameArgs[0], hostnameArgs[1:]...); err != nil {
			util.Warning("Failed to execute %s, you will need to manually execute '%s' with administrative privileges, err=%v, output=%v", strings.Join(hostnameArgs, " "), strings.Join(hostnameArgs, " "), err, out)
		}
	}

	return nil
}

// CheckWindowsHostsFile() verifies that the Windows hosts file doesn't have long lines in it.
func CheckWindowsHostsFile() {
	if runtime.GOOS != "windows" {
		return
	}
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		util.Warning("unable to GetDockerIP(): %v", err)
	}
	hosts, err := ddevhosts.New()
	if err != nil {
		util.Warning("could not open hostfile: %v", err)
	}

	ipPosition := hosts.GetIPPosition(dockerIP)
	if ipPosition != -1 {
		hostsLine := hosts.Lines[ipPosition]
		if len(hostsLine.Hosts) >= 10 {
			util.Error("You have more than 9 entries in your (windows) hostsfile entry for %s", dockerIP)
			util.Error("Please use `ddev hostname --remove-inactive` or edit the hosts file manually")
			util.Error("Please see %s for more information", "https://ddev.readthedocs.io/en/stable/users/basics/troubleshooting/#windows-hosts-file-limited-to-10-hosts-per-ip-address-line")
		}
	}
}
