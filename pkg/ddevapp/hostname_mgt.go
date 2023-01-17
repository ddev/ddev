package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevhosts"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	goodhosts "github.com/goodhosts/hostsfile"
	"net"
	"os"
	exec2 "os/exec"
	"strings"
)

// windowsDdevExeAvailable says if ddev.exe is available on Windows side
var windowsDdevExeAvailable bool

// IsWindowsDdevExeAvailable checks to see if we can use ddev.exe on Windows side
func IsWindowsDdevExeAvailable() bool {
	if !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt && !windowsDdevExeAvailable && dockerutil.IsWSL2() {
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

func IsHostnameInHostsFile(hostname string) (bool, error) {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return false, fmt.Errorf("could not get Docker IP: %v", err)
	}

	var hosts = &ddevhosts.DdevHosts{}
	if dockerutil.IsWSL2() && !globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt {
		hosts, err = ddevhosts.NewCustomHosts(ddevhosts.WSL2WindowsHostsFile)
	} else {
		hosts, err = ddevhosts.New()
	}
	if err != nil {
		return false, fmt.Errorf("Unable to open hosts file: %v", err)
	}
	return hosts.Has(dockerIP, hostname), nil
}

// AddHostsEntriesIfNeeded will (optionally) add the site URL to the host's /etc/hosts.
func (app *DdevApp) AddHostsEntriesIfNeeded() error {
	var err error
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

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
		exists, err := IsHostnameInHostsFile(name)
		if exists {
			continue
		}
		if err != nil {
			util.Warning("unable to open hosts file: %v", err)
			continue
		}
		util.Warning("The hostname %s is not currently resolvable, trying to add it to the hosts file", name)
		if !dockerutil.IsWSL2() || globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt {
			err = AddHostEntry(name, dockerIP)
		} else {
			if IsWindowsDdevExeAvailable() {
				err = WSL2AddHostEntry(name, dockerIP)
			} else {
				util.Warning("ddev.exe is not available on the Windows side. Please install it with 'choco install -y ddev' or disable Windows-side hosts management using 'ddev config global --wsl2-no-windows-hosts-mgt'")
				err = nil
			}
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// AddHostEntry adds an entry to default hosts file
// This version is NOT used on WSL2
// We would have hoped to use DNS or have found the entry already in hosts
// But if it's not, try to add one.
func AddHostEntry(name string, ip string) error {
	_, err := exec2.LookPath("sudo")
	if (os.Getenv("DDEV_NONINTERACTIVE") != "") || err != nil {
		util.Warning("You must manually add the following entry to your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname %s %s'", ip, name, name, ip)
		return nil
	}

	ddevFullpath, err := os.Executable()
	util.CheckErr(err)

	hostnameArgs := []string{ddevFullpath, "hostname", name, ip}
	output.UserOut.Printf("ddev needs to add an entry to your hostfile.\nIt may require administrative privileges via the sudo command, so you may be required\nto enter your password for sudo or allow escalation. ddev is about to issue the command:\n   %s", strings.Join(hostnameArgs, " "))

	output.UserOut.Println("Please enter your password or allow escalation if prompted.")
	out := ""
	out, err = runCommandWithSudo(hostnameArgs)
	if err != nil {
		util.Warning("Failed to execute %s, you will need to manually execute '%s' with administrative privileges, err=%v, output=%v", strings.Join(hostnameArgs, " "), strings.Join(hostnameArgs, " "), err, out)
		return err
	}
	util.Success(out)
	return nil
}

// WSL2AddHostEntry adds a hosts file entry on the Windows side using sudo and ddev.exe
func WSL2AddHostEntry(name string, ip string) error {
	hostnameArgs := []string{"sudo.exe", "ddev.exe", "hostname", name, ip}
	output.UserOut.Printf("ddev needs to add an entry to your Windows hosts file.\nIt may require escalation. ddev is about to issue the command:\n   %s", strings.Join(hostnameArgs, " "))
	out, err := exec.RunHostCommand(hostnameArgs[0], hostnameArgs[1:]...)
	if err != nil {
		util.Warning("Failed to execute %s, you will need to manually execute '%s' with administrative privileges, err=%v, output=%v", strings.Join(hostnameArgs, " "), strings.Join(hostnameArgs, " "), err, out)
	}
	util.Success(out)
	return nil
}

// RemoveHostsEntriesIfNeeded will remove the site URL from the host's /etc/hosts.
func (app *DdevApp) RemoveHostsEntriesIfNeeded() error {
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	for _, name := range app.GetHostnames() {
		checkName := strings.TrimPrefix(name, "*.")
		hostIPs, err := net.LookupHost(checkName)

		// If failed lookup, or more than one IP, or IP doesn't match
		// take no action
		if err != nil || (len(hostIPs) > 0 && hostIPs[0] != dockerIP) {
			continue
		}

		// We likely won't hit the hosts.Has() as true because
		// we already did a lookup. But check anyway.
		exists, err := IsHostnameInHostsFile(name)
		if !exists {
			continue
		}
		if err != nil {
			util.Warning("unable to open hosts file: %v", err)
			continue
		}

		if !dockerutil.IsWSL2() || globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt {
			err = RemoveHostEntry(name, dockerIP)
		} else {
			if IsWindowsDdevExeAvailable() {
				err = WSL2RemoveHostEntry(name, dockerIP)
			} else {
				util.Warning("ddev.exe is not available on the Windows side. Please install it with 'choco install -y ddev' or disable Windows-side hosts management using 'ddev config global --wsl2-no-windows-hosts-mgt'")
			}
		}
		if err != nil {
			util.Warning("Failed to remove host entry %s: %v", name, err)
		}
	}

	return nil
}

// RemoveHostEntry removes named /etc/hosts entry if it exists
// This version is used everywhere except wsl2, which has its own approach
func RemoveHostEntry(name string, dockerIP string) error {

	hosts, err := goodhosts.NewHosts()
	if err != nil {
		return err
	}

	// If the hosts file doesn't contain the hostname in the first place,
	// return.
	if !hosts.Has(dockerIP, name) {
		return nil
	}

	ddevFullPath, err := os.Executable()
	if err != nil {
		return err
	}
	_, err = exec2.LookPath("sudo")
	if os.Getenv("DDEV_NONINTERACTIVE") != "" || err != nil {
		util.Warning("You must manually remove the following entry from your hosts file:\n%s %s\nOr with root/administrative privileges execute 'ddev hostname --remove %s %s", dockerIP, name, name, dockerIP)
		return nil
	}

	hostnameArgs := []string{"sudo", ddevFullPath, "hostname", "--remove", name, dockerIP}
	command := strings.Join(hostnameArgs, " ")
	util.Warning(fmt.Sprintf("    sudo %s", command))
	output.UserOut.Printf("ddev will remove '%s' from your hosts file.\nIt may require administrative privileges via the sudo command or Windows UAC, so you may be required\nto enter your sudo password. ddev is about to issue the command:\n    %s", name, strings.Join(hostnameArgs, " "))
	output.UserOut.Println("Please enter your password if prompted.")

	out := ""
	if out, err = exec.RunHostCommand(hostnameArgs[0], hostnameArgs[1:]...); err != nil {
		util.Warning("Failed to execute %s, you will need to manually execute '%s' with administrative privileges, err=%v, output=%v", strings.Join(hostnameArgs, " "), strings.Join(hostnameArgs, " "), err, out)
		return err
	}
	util.Success(out)
	return nil
}

// WSL2RemoveHostEntry uses Windows-side sudo.exe ddev.exe to remove from hosts file, with UAC escalation
func WSL2RemoveHostEntry(name string, ip string) error {
	hostnameArgs := []string{"sudo.exe", "ddev.exe", "hostname", "--remove", name, ip}
	output.UserOut.Printf("ddev needs to remove an entry from your Windows hosts file.\nIt may require UAC escalation. ddev is about to issue the command:\n   %s", strings.Join(hostnameArgs, " "))
	out, err := exec.RunHostCommand(hostnameArgs[0], hostnameArgs[1:]...)
	if err != nil {
		util.Warning("Failed to execute %s, you will need to manually execute '%s' with administrative privileges, err=%v, output=%v", strings.Join(hostnameArgs, " "), strings.Join(hostnameArgs, " "), err, out)
		return err
	}
	util.Success(out)
	return nil
}

// runCommandWithSudo adds sudo to command if we aren't already running with root privs
func runCommandWithSudo(args []string) (out string, err error) {
	if err != nil {
		return "", fmt.Errorf("could not get home directory for current user. is it set?")
	}

	if os.Geteuid() != 0 {
		args = append([]string{"sudo", "--preserve-env=HOME"}, args...)
	}
	out, err = exec.RunHostCommand(args[0], args[1:]...)
	return out, err
}
