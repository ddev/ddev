package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/output"

	"strings"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/version"
	"github.com/lextoumbourou/goodhosts"
	"github.com/spf13/cobra"
)

var removeHostName bool
var removeInactive bool

// HostNameCmd represents the hostname command
var HostNameCmd = &cobra.Command{
	Use:   "hostname [hostname] [ip]",
	Short: "Manage your hostfile entries.",
	Long: `Manage your hostfile entries. Managing host names has security and usability
implications and requires elevated privileges. You may be asked for a password
to allow ddev to modify your hosts file.`,
	Run: func(cmd *cobra.Command, args []string) {
		hosts, err := goodhosts.NewHosts()
		if err != nil {
			rawResult := make(map[string]interface{})
			detail := fmt.Sprintf("Could not open hosts file for reading: %v", err)
			rawResult["error"] = "READERROR"
			rawResult["full_error"] = detail
			output.UserOut.WithField("raw", rawResult).Fatal(detail)

			return
		}

		// Attempt to write the hosts file first to catch any permissions issues early
		if err := hosts.Flush(); err != nil {
			rawResult := make(map[string]interface{})
			detail := fmt.Sprintf("Please use sudo or execute with administrative privileges: %v", err)
			rawResult["error"] = "WRITEERROR"
			rawResult["full_error"] = detail
			output.UserOut.WithField("raw", rawResult).Fatal(detail)

			return
		}

		// If requested, remove all inactive host names and exit
		if removeInactive {
			if len(args) > 0 {
				output.UserOut.Fatal("Invalid arguments supplied. 'ddev hostname --remove-all' accepts no arguments.")
			}

			removeInactiveHostnames(hosts)

			return
		}

		// If operating on one host name, two arguments are required
		if len(args) != 2 {
			output.UserOut.Fatal("Invalid arguments supplied. Please use 'ddev hostname [hostname] [ip]'")
		}

		hostname, ip := args[0], args[1]

		// If requested, remove the provided host name and exit
		if removeHostName {
			removeHostname(hosts, ip, hostname)

			return
		}

		// By default, add a host name
		addHostname(hosts, ip, hostname)
	},
}

// addHostname encapsulates the logic of adding a hostname to the system's hosts file.
func addHostname(hosts goodhosts.Hosts, ip, hostname string) {
	var detail string
	rawResult := make(map[string]interface{})

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

	detail = "Hostname added to hosts file"
	rawResult["error"] = "SUCCESS"
	rawResult["detail"] = detail
	output.UserOut.WithField("raw", rawResult).Info(detail)

	return
}

// removeHostname encapsulates the logic of removing a hostname from the system's hosts file.
func removeHostname(hosts goodhosts.Hosts, ip, hostname string) {
	var detail string
	rawResult := make(map[string]interface{})

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

	detail = "Hostname removed from hosts file"
	rawResult["error"] = "SUCCESS"
	rawResult["detail"] = detail
	output.UserOut.WithField("raw", rawResult).Info(detail)

	return
}

// removeInactiveHostnames will remove all host names except those current in use by active projects.
func removeInactiveHostnames(hosts goodhosts.Hosts) {
	var detail string
	rawResult := make(map[string]interface{})

	// Get the list active hosts names to preserve
	activeHostNames := make(map[string]bool)
	for _, app := range ddevapp.GetApps() {
		for _, h := range app.GetHostnames() {
			activeHostNames[h] = true
		}
	}

	// Find all current host names for the local IP address
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		detail = fmt.Sprintf("Failed to get Docker IP: %v", err)
		rawResult["error"] = "DOCKERERROR"
		rawResult["full_error"] = detail
		output.UserOut.WithField("raw", rawResult).Fatal(detail)
	}

	// Iterate through each host line
	for _, line := range hosts.Lines {
		// Checking if it concerns the local IP address
		if line.IP == dockerIP {
			// Iterate through each registered host
			for _, h := range line.Hosts {
				internalResult := make(map[string]interface{})

				// Ignore those we want to preserve
				if isActiveHost := activeHostNames[h]; isActiveHost {
					detail = fmt.Sprintf("Hostname %s at %s is active, preserving", h, line.IP)
					internalResult["error"] = "SUCCESS"
					internalResult["detail"] = detail
					output.UserOut.WithField("raw", internalResult).Info(detail)
					continue
				}

				// Silently ignore those that may not be ddev-managed
				if !strings.HasSuffix(h, version.DDevTLD) {
					continue
				}

				// Remaining host names are fair game to be removed
				if err := hosts.Remove(line.IP, h); err != nil {
					detail = fmt.Sprintf("Could not remove hostname %s at %s: %v", h, line.IP, err)
					internalResult["error"] = "REMOVEERROR"
					internalResult["full_error"] = detail
					output.UserOut.WithField("raw", internalResult).Fatal(detail)
				}

				detail = fmt.Sprintf("Removed hostname %s at %s", h, line.IP)
				internalResult["error"] = "SUCCESS"
				internalResult["detail"] = detail
				output.UserOut.WithField("raw", internalResult).Info(detail)
			}
		}
	}

	if err := hosts.Flush(); err != nil {
		detail = fmt.Sprintf("Could not write hosts file: %v", err)
		rawResult["error"] = "WRITEERROR"
		rawResult["full_error"] = detail
		output.UserOut.WithField("raw", rawResult).Fatal(detail)
	}

	return
}

func init() {
	HostNameCmd.Flags().BoolVarP(&removeHostName, "remove", "r", false, "Remove the provided host name - ip correlation")
	HostNameCmd.Flags().BoolVarP(&removeInactive, "remove-inactive", "R", false, "Remove host names of inactive projects")
	HostNameCmd.Flags().BoolVar(&removeInactive, "fire-bazooka", false, "Alias of --remove-inactive")
	RootCmd.AddCommand(HostNameCmd)
}
