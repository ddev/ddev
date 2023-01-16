package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	goodhosts "github.com/goodhosts/hostsfile"
	"github.com/spf13/cobra"
	"os"
)

var removeHostnameFlag bool
var removeInactiveFlag bool
var checkHostnameFlag bool

// HostNameCmd represents the hostname command
var HostNameCmd = &cobra.Command{
	Use:   "hostname [hostname] [ip]",
	Short: "Manage your hostfile entries.",
	Example: `
ddev hostname junk.example.com 127.0.0.1
ddev hostname -r junk.example.com 127.0.0.1
ddev hostname --check junk.example.com 127.0.0.1
ddev hostname --remove-inactive
`,
	Long: `Manage your hostfile entries. Managing host names has security and usability
implications and requires elevated privileges. You may be asked for a password
to allow ddev to modify your hosts file. If you are connected to the internet and using the domain ddev.site this is generally not necessary, because the hosts file never gets manipulated.`,
	Run: func(cmd *cobra.Command, args []string) {

		// If requested, remove all inactive host names and exit
		if removeInactiveFlag {
			if len(args) > 0 {
				util.Failed("Invalid arguments supplied. 'ddev hostname --remove-all' accepts no arguments.")
			}

			util.Warning("Attempting to remove inactive custom hostnames for projects which are registered but not running")
			removeInactiveHostnames()
			return
		}

		// If operating on one host name, two arguments are required
		if len(args) != 2 {
			util.Failed("Invalid arguments supplied. Please use 'ddev hostname [hostname] [ip]'")
		}

		name, dockerIP := args[0], args[1]
		var err error

		// If requested, remove the provided host name and exit
		if removeHostnameFlag {
			if !dockerutil.IsWSL2() || globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt {
				err = ddevapp.RemoveHostEntry(name, dockerIP)
			} else {
				if ddevapp.IsWindowsDdevExeAvailable() {
					err = ddevapp.WSL2RemoveHostEntry(name, dockerIP)
				} else {
					util.Warning("ddev.exe is not available on the Windows side. Please install it with 'choco install -y ddev' or disable Windows-side hosts management using 'ddev config global --wsl2-no-windows-hosts-mgt'")
				}
			}
			if err != nil {
				util.Warning("Failed to remove host entry %s: %v", name, err)
			}
			return
		}
		if checkHostnameFlag {
			exists, err := ddevapp.IsHostnameInHostsFile(name)
			if exists {
				return
			}
			if err != nil {
				util.Warning("could not check existence in hosts file: %v", err)
			}
			os.Exit(1)
		}
		// By default, add a host name
		if !dockerutil.IsWSL2() || globalconfig.DdevGlobalConfig.WSL2NoWindowsHostsMgt {
			err = ddevapp.AddHostEntry(name, dockerIP)
		} else {
			if ddevapp.IsWindowsDdevExeAvailable() {
				err = ddevapp.WSL2AddHostEntry(name, dockerIP)
			} else {
				util.Warning("ddev.exe is not available on the Windows side. Please install it with 'choco install -y ddev' or disable Windows-side hosts management using 'ddev config global --wsl2-no-windows-hosts-mgt'")
			}
		}
		if err != nil {
			util.Warning("Failed to remove add hosts entry %s: %v", name, err)
		}
	},
}

// addHostname encapsulates the logic of adding a hostname to the system's hosts file.
func addHostname(hosts *goodhosts.Hosts, ip, hostname string) {
	var detail string
	rawResult := make(map[string]interface{})

	if dockerutil.IsWSL2() && ddevapp.IsWindowsDdevExeAvailable() {
		util.Debug("Running sudo.exe ddev.exe %s %s  on Windows side", hostname, ip)
		out, err := exec.RunHostCommand("sudo.exe", "ddev.exe", "hostname", hostname, ip)
		if err == nil {
			util.Debug("ran sudo.exe ddev.exe %s %s with output=%s", hostname, ip, out)
			return
		}
		util.Warning("Unable to run sudo.exe ddev.exe hostname %s %s on Windows side, continuing with WSL2 /etc/hosts err=%s, output=%s", hostname, ip, err, out)
	}

	if hosts.Has(ip, hostname) {
		detail = "Hostname already exists in hosts file"
		rawResult["error"] = "SUCCESS"
		rawResult["detail"] = detail
		output.UserOut.WithField("raw", rawResult).Info(detail)

		return
	}

	if err := hosts.Add(ip, hostname); err != nil {
		detail = fmt.Sprintf("Could not add hostname %s at %s: %v", hostname, ip, err)
		rawResult["error"] = "ADDERROR"
		rawResult["full_error"] = detail
		output.UserOut.WithField("raw", rawResult).Fatal(detail)

		return
	}

	if err := hosts.Flush(); err != nil {
		detail = fmt.Sprintf("Could not write hosts file: %v", err)
		rawResult["error"] = "WRITEERROR"
		rawResult["full_error"] = detail
		output.UserOut.WithField("raw", rawResult).Fatal(detail)

		return
	}

	detail = fmt.Sprintf("Hostname '%s' added to hosts file", hostname)
	rawResult["error"] = "SUCCESS"
	rawResult["detail"] = detail
	output.UserOut.WithField("raw", rawResult).Info(detail)

	return
}

// removeHostname encapsulates the logic of removing a hostname from the system's hosts file.
func removeHostname(hosts *goodhosts.Hosts, ip, hostname string) {
	var detail string
	rawResult := make(map[string]interface{})

	if dockerutil.IsWSL2() && ddevapp.IsWindowsDdevExeAvailable() {
		util.Debug("Running ddev.exe --check %s %s  on Windows side", hostname, ip)
		out, err := exec.RunHostCommand("ddev.exe", "hostname", "--check", hostname, ip)
		if err != nil {
			util.Debug("ddev.exe --check hostname says hostname doesn't exist on windows; ran %s %s with output=%s, err=%v", hostname, ip, out, err)
			return
		}
		util.Debug("Running sudo.exe ddev.exe -r %s %s  on Windows side", hostname, ip)
		out, err = exec.RunHostCommand("sudo.exe", "ddev.exe", "hostname", "--remove", hostname, ip)
		if err == nil {
			util.Debug("ran sudo.exe ddev.exe --remove %s %s with output=%s", hostname, ip, out)
			return
		}
		util.Warning("Unable to run sudo.exe ddev.exe hostname --remove %s %s on Windows side, continuing with WSL2 /etc/hosts err=%s, output=%s", hostname, ip, err, out)
	}

	if !hosts.Has(ip, hostname) {
		detail = "Hostname does not exist in hosts file"
		rawResult["error"] = "SUCCESS"
		rawResult["detail"] = detail
		output.UserOut.WithField("raw", rawResult).Info(detail)

		return
	}

	if err := hosts.Remove(ip, hostname); err != nil {
		detail = fmt.Sprintf("Could not remove hostname %s at %s: %v", hostname, ip, err)
		rawResult["error"] = "REMOVEERROR"
		rawResult["full_error"] = detail
		output.UserOut.WithField("raw", rawResult).Fatal(detail)

		return
	}

	if err := hosts.Flush(); err != nil {
		detail = fmt.Sprintf("Could not write hosts file: %v", err)
		rawResult["error"] = "WRITEERROR"
		rawResult["full_error"] = detail
		output.UserOut.WithField("raw", rawResult).Fatal(detail)

		return
	}

	detail = fmt.Sprintf("Hostname '%s' removed from hosts file", hostname)
	rawResult["error"] = "SUCCESS"
	rawResult["detail"] = detail
	output.UserOut.WithField("raw", rawResult).Info(detail)

	return
}

// checkHostname checks to see if hostname already exists in hosts file.
func checkHostname(hosts *goodhosts.Hosts, ip, hostname string) bool {
	if dockerutil.IsWSL2() && ddevapp.IsWindowsDdevExeAvailable() {
		util.Debug("Running ddev.exe --check %s %s  on Windows side", hostname, ip)
		out, err := exec.RunHostCommand("ddev.exe", "hostname", "--check", hostname, ip)
		if err == nil {
			util.Debug("ran ddev.exe --check %s %s with output=%s", hostname, ip, out)
			return true
		}
		return false
	}

	return hosts.Has(ip, hostname)
}

// removeInactiveHostnames will remove all host names except those current in use by active projects.
func removeInactiveHostnames() {
	apps, err := ddevapp.GetInactiveProjects()
	if err != nil {
		util.Warning("unable to run GetInactiveProjects: %v", err)
		return
	}
	for _, app := range apps {
		err := app.RemoveHostsEntriesIfNeeded()
		if err != nil {
			util.Warning("unable to remove hosts entries for project '%s': %v", app.Name, err)
		}
	}
	return
}

func init() {
	HostNameCmd.Flags().BoolVarP(&removeHostnameFlag, "remove", "r", false, "Remove the provided host name - ip correlation")
	HostNameCmd.Flags().BoolVarP(&checkHostnameFlag, "check", "c", false, "Check to see if provided hostname is already in hosts file")
	HostNameCmd.Flags().BoolVarP(&removeInactiveFlag, "remove-inactive", "R", false, "Remove host names of inactive projects")
	HostNameCmd.Flags().BoolVar(&removeInactiveFlag, "fire-bazooka", false, "Alias of --remove-inactive")
	_ = HostNameCmd.Flags().MarkHidden("fire-bazooka")

	RootCmd.AddCommand(HostNameCmd)
}
