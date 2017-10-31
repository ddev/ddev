package cmd

import (
	"log"

	"github.com/lextoumbourou/goodhosts"
	"github.com/spf13/cobra"
)

// HostNameCmd represents the local command
var HostNameCmd = &cobra.Command{
	Use:   "hostname [ip] [hostnames]",
	Short: "Manage your hostfile entries.",
	Long:  `Manage your hostfile entries.`,
	Run: func(cmd *cobra.Command, args []string) {
		ip, hostnames := args[0], args[1:]

		hosts, err := goodhosts.NewHosts()
		if err != nil {
			log.Fatalf("failed to open hosts file: %v", err)
		}

		for i, host := range hostnames {
			if hosts.Has(ip, host) {
				hostnames = append(hostnames[:i], hostnames[i+1:]...)
			}
		}

		if len(hostnames) > 0 {
			err = hosts.Add(ip, hostnames...)
			if err != nil {
				log.Fatalf("failed to add hostname: %v", err)
			}

			if err := hosts.Flush(); err != nil {
				log.Fatalf("failed to write hosts file: %v", err)
			}
		}
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
}

func init() {
	RootCmd.AddCommand(HostNameCmd)
}
