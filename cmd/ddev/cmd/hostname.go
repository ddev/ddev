package cmd

import (
	"github.com/drud/ddev/pkg/output"
	log "github.com/sirupsen/logrus"

	"github.com/drud/ddev/pkg/util"
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

		hostname, ip := args[0], args[1]
		hosts, err := goodhosts.NewHosts()
		if err != nil {
			util.Failed("could not open hosts file. %s", err)
		}
		if hosts.Has(ip, hostname) {
			log.Debugf("Hosts file entry %s exists, taking no action", hostname)
			return
		}

		err = hosts.Add(ip, hostname)
		if err != nil {
			util.Failed("Could not add hostname")
		}

		if err := hosts.Flush(); err != nil {
			util.Failed("Could not write hosts file: %s", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(HostNameCmd)
}
