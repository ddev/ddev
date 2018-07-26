package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/output"

	"github.com/lextoumbourou/goodhosts"
	"github.com/spf13/cobra"
)

var removeHostName bool

// HostNameCmd represents the hostname command
var HostNameCmd = &cobra.Command{
	Use:   "hostname [hostname] [ip]",
	Short: "Manage your hostfile entries.",
	Long:  `Manage your hostfile entries.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			output.UserOut.Fatal("Invalid arguments supplied. Please use 'ddev hostname [hostname] [ip]'")
		}

		hostname, ip := args[0], args[1]

		hosts, err := goodhosts.NewHosts()
		if err != nil {
			detail := fmt.Sprintf("Could not open hosts file for reading: %v", err)
			rawResult := make(map[string]interface{})
			rawResult["error"] = "READERROR"
			rawResult["full_error"] = detail
			output.UserOut.WithField("raw", rawResult).Fatal(detail)

			return
		}

		if removeHostName {
			removeHostname(hosts, ip, hostname)

			return
		}

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
		detail = fmt.Sprintf("Could notk add hostname %s at %s: %v", hostname, ip, err)
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
	rawResult := make(map[string]interface{})

	if !hosts.Has(ip, hostname) {
		detail := "Hostname does not exist in hosts file"
		rawResult["error"] = "SUCCESS"
		rawResult["detail"] = detail
		output.UserOut.WithField("raw", rawResult).Info(detail)

		return
	}

	if err := hosts.Remove(ip, hostname); err != nil {
		detail := fmt.Sprintf("Could not remove hostname %s at %s: %v", hostname, ip, err)
		rawResult["error"] = "REMOVEERROR"
		rawResult["full_error"] = detail
		output.UserOut.WithField("raw", rawResult).Fatal(detail)

		return
	}

	if err := hosts.Flush(); err != nil {
		detail := fmt.Sprintf("Could not write hosts file: %v", err)
		rawResult["error"] = "WRITEERROR"
		rawResult["full_error"] = detail
		output.UserOut.WithField("raw", rawResult).Info(detail)

		return
	}

	detail := "Hostname removed from hosts file"
	rawResult["error"] = "SUCCESS"
	rawResult["detail"] = detail
	output.UserOut.WithField("raw", rawResult).Info(detail)

	return
}

func init() {
	HostNameCmd.Flags().BoolVarP(&removeHostName, "remove", "R", false, "Remove the provided host name - ip correlation")
	RootCmd.AddCommand(HostNameCmd)
}
