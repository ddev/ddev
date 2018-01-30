package cmd

import (
	"fmt"

	"github.com/drud/ddev/pkg/output"

	"github.com/lextoumbourou/goodhosts"
	"github.com/spf13/cobra"
)

// HostNameCmd represents the hostname command
var HostNameCmd = &cobra.Command{
	Use:   "hostname [hostname] [ip]",
	Short: "Manage your hostfile entries.",
	Long:  `Manage your hostfile entries.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			output.UserOut.Fatal("Invalid arguments supplied. Please use 'ddev hostname [hostname] [ip]'")
		}

		rawResult := make(map[string]interface{})

		hostname, ip := args[0], args[1]
		hosts, err := goodhosts.NewHosts()
		if err != nil {
			rawResult["error"] = "READERROR"
			rawResult["full_error"] = fmt.Sprintf("%v", err)
			output.UserOut.WithField("raw", rawResult).Fatal(fmt.Sprintf("could not open hosts file for read: %v", err))
			return
		}
		if hosts.Has(ip, hostname) {
			if output.JSONOutput {
				rawResult["error"] = "SUCCESS"
				rawResult["detail"] = "hostname already exists in hosts file"
				output.UserOut.WithField("raw", rawResult).Info("")
			}
			return
		}

		err = hosts.Add(ip, hostname)
		if err != nil {
			rawResult["error"] = "ADDERROR"
			rawResult["full_error"] = fmt.Sprintf("%v", err)
			output.UserOut.WithField("raw", rawResult).Fatal(fmt.Sprintf("could not add hostname %s at %s: %v", hostname, ip, err))
		}

		if err := hosts.Flush(); err != nil {
			rawResult["error"] = "WRITEERROR"
			rawResult["full_error"] = fmt.Sprintf("%v", err)
			output.UserOut.WithField("raw", rawResult).Fatal(fmt.Sprintf("Could not write hosts file: %v", err))
		} else {
			rawResult["error"] = "SUCCESS"
			rawResult["detail"] = "hostname added to hosts file"
			output.UserOut.WithField("raw", rawResult).Info("")
		}
	},
}

func init() {
	RootCmd.AddCommand(HostNameCmd)
}
