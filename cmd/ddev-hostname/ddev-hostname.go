package main

import (
	"github.com/ddev/ddev/pkg/hostname"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	_ = RootCmd.Execute()
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

var removeHostnameFlag bool
var checkHostnameFlag bool

// RootCmd is the ddev-hostname command
var RootCmd = &cobra.Command{
	Use:     "ddev-hostname [hostname] [ip]",
	Short:   "Manage your hostfile entries.",
	Version: versionconstants.DdevVersion,
	Example: `
ddev-hostname junk.example.com 127.0.0.1
ddev-hostname -r junk.example.com 127.0.0.1
ddev-hostname --check junk.example.com 127.0.0.1
`,
	Long: `Manage your hostfile entries. Managing host names has security and usability
implications and requires elevated privileges. You may be asked for a password
to allow ddev-hostname to modify your hosts file. If you are connected to the internet and using the domain ddev.site this is generally not necessary, because the hosts file never gets manipulated.`,
	Run: func(_ *cobra.Command, args []string) {
		// If operating on one host name, two arguments are required
		if len(args) != 2 {
			util.Failed("Invalid arguments supplied. Please use 'ddev-hostname [hostname] [ip]'")
		}

		name, dockerIP := args[0], args[1]

		inHostsFile, err := hostname.IsHostnameInHostsFile(name)
		if err != nil {
			util.Warning("Could not check existence of %s in hosts file: %v", name, err)
		}

		// If requested, remove the provided host name and exit
		if removeHostnameFlag {
			if inHostsFile {
				util.Debug("Elevating privileges to remove host entry %s -> %s", name, dockerIP)
				elevateIfNeeded()
				err := hostname.RemoveHostEntry(name, dockerIP)
				if err != nil {
					util.Warning("Failed to remove host entry %s: %v", name, err)
				}
			}
			return
		}
		if checkHostnameFlag {
			if inHostsFile {
				return
			}
			os.Exit(1)
		}
		// By default, add a host name
		if !inHostsFile {
			util.Debug("Elevating privileges to add host entry %s -> %s", name, dockerIP)
			elevateIfNeeded()
			err = hostname.AddHostEntry(name, dockerIP)
			if err != nil {
				util.Warning("Failed to add hosts entry %s:%s: %v", name, dockerIP, err)
			}
		}
	},
}

func init() {
	RootCmd.Flags().BoolVarP(&removeHostnameFlag, "remove", "r", false, "Remove the provided host name - ip correlation")
	RootCmd.Flags().BoolVarP(&checkHostnameFlag, "check", "c", false, "Check to see if provided hostname is already in hosts file")
}
