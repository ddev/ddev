package cmd

import (
	"fmt"
	"log"

	"github.com/lextoumbourou/goodhosts"
	"github.com/spf13/cobra"
)

// HostNameCmd represents the local command
var HostNameCmd = &cobra.Command{
	Use:   "hostname [hostname] [ip]",
	Short: "Local dev for legacy sites.",
	Long:  `Manage your hostfile entries..`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			log.Fatal("Invalid arguments supplied. Please use 'drud legacy hostname [hostname] [ip]'")
		}

		hostname, ip := args[0], args[1]
		hosts, err := goodhosts.NewHosts()
		if err != nil {
			log.Fatalf("could not open hostfile. %s", err)
		}
		if hosts.Has(ip, hostname) {
			fmt.Println("Entry exists!")
			return
		}

		err = hosts.Add(ip, hostname)
		if err != nil {
			log.Fatal("Could not add hostname")
		}

		if err := hosts.Flush(); err != nil {
			log.Fatalf("could not write hosts file: %s", err)
		}
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
}

func init() {
	LegacyCmd.AddCommand(HostNameCmd)
}
