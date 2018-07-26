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
			rawResult := make(map[string]interface{})
			rawResult["error"] = "READERROR"
			rawResult["full_error"] = fmt.Sprintf("%v", err)
			output.UserOut.WithField("raw", rawResult).Fatal(fmt.Sprintf("could not open hosts file for read: %v", err))

			return
		}

		if removeHostName {
			removeHost(hosts, ip, hostname)

			return
		}

		addHost(hosts, ip, hostname)
	},
}

func addHost(hosts goodhosts.Hosts, ip, hostname string) {
	rawResult := make(map[string]interface{})

	if hosts.Has(ip, hostname) {
		if output.JSONOutput {
			rawResult["error"] = "SUCCESS"
			rawResult["detail"] = "hostname already exists in hosts file"
			output.UserOut.WithField("raw", rawResult).Info("")
		}

		return
	}

	if err := hosts.Add(ip, hostname); err != nil {
		rawResult["error"] = "ADDERROR"
		rawResult["full_error"] = fmt.Sprintf("%v", err)
		output.UserOut.WithField("raw", rawResult).Fatal(fmt.Sprintf("could not add hostname %s at %s: %v", hostname, ip, err))

		return
	}

	if err := hosts.Flush(); err != nil {
		rawResult["error"] = "WRITEERROR"
		rawResult["full_error"] = fmt.Sprintf("%v", err)
		output.UserOut.WithField("raw", rawResult).Fatal(fmt.Sprintf("Could not write hosts file: %v", err))

		return
	}

	rawResult["error"] = "SUCCESS"
	rawResult["detail"] = "hostname added to hosts file"
	output.UserOut.WithField("raw", rawResult).Info("")

	return
}

func removeHost(hosts goodhosts.Hosts, ip, hostname string) {
	rawResult := make(map[string]interface{})

	if !hosts.Has(ip, hostname) {
		rawResult["error"] = "SUCCESS"
		rawResult["detail"] = "hostname does not exist in hosts file"
		output.UserOut.WithField("raw", rawResult).Info("")

		return
	}

	if err := hosts.Remove(ip, hostname); err != nil {
		rawResult["error"] = "REMOVEERROR"
		rawResult["full_error"] = fmt.Sprintf("%v", err)
		output.UserOut.WithField("raw", rawResult).Fatal(fmt.Sprintf("could not remove hostname %s at %s: %v", hostname, ip, err))

		return
	}

	if err := hosts.Flush(); err != nil {
		rawResult["error"] = "WRITERROR"
		rawResult["full_error"] = fmt.Sprintf("%v", err)
		output.UserOut.WithField("raw", rawResult).Info("")

		return
	}

	rawResult["error"] = "SUCCESS"
	rawResult["detail"] = "hostname removed from hosts file"
	output.UserOut.WithField("raw", rawResult).Info("")

	return
}

func init() {
	HostNameCmd.Flags().BoolVarP(&removeHostName, "remove", "R", false, "Remove the provided host name - ip correlation")
	RootCmd.AddCommand(HostNameCmd)
}
