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

// RootCmd is the ddev_hostname command
var RootCmd = &cobra.Command{
	Use:     "ddev_hostname [hostname] [ip]",
	Short:   "Manage your hostfile entries.",
	Version: versionconstants.DdevVersion,
	Example: `
ddev_hostname junk.example.com 127.0.0.1
ddev_hostname -r junk.example.com 127.0.0.1
ddev_hostname --check junk.example.com 127.0.0.1
`,
	Long: `Manage your hostfile entries. Managing host names has security and usability
implications and requires elevated privileges. You may be asked for a password
to allow ddev_hostname to modify your hosts file. If you are connected to the internet and using the domain ddev.site this is generally not necessary, because the hosts file never gets manipulated.`,
	Run: func(_ *cobra.Command, args []string) {

		// TODO: Fix up for DDEV_NONINTERACTIVE- I think that should probably be done in the caller.
		// Unless DDEV_NONINTERACTIVE is set (tests) then we need to be admin
		//if os.Getenv("DDEV_NONINTERACTIVE") == "" && os.Geteuid() != 0 && !checkHostnameFlag && !removeInactiveFlag && runtime.GOOS != "windows" {
		//	util.Failed("'ddev hostname %s' must be run with administrator privileges", strings.Join(args, " "))
		//}

		// TODO: Reimplement this, figure out how to know what's inactive
		// This may not be useful and not need to be implemented.
		// If requested, remove all inactive host names and exit
		//if removeInactiveFlag {
		//	if len(args) > 0 {
		//		util.Failed("Invalid arguments supplied. 'ddev hostname --remove-all' accepts no arguments.")
		//	}
		//
		//	util.Warning("Attempting to remove inactive custom hostnames for projects which are registered but not running")
		//	//removeInactiveHostnames()
		//	return
		//}

		// If operating on one host name, two arguments are required
		if len(args) != 2 {
			util.Failed("Invalid arguments supplied. Please use 'ddev_hostname [hostname] [ip]'")
		}

		name, dockerIP := args[0], args[1]
		var err error

		util.Debug("Elevating privileges to add host entry %s -> %s", name, dockerIP)
		elevateIfNeeded()

		// If requested, remove the provided host name and exit
		if removeHostnameFlag {
			err = hostname.RemoveHostEntry(name, dockerIP)
			if err != nil {
				util.Warning("Failed to remove host entry %s: %v", name, err)
			}
			return
		}
		if checkHostnameFlag {
			exists, err := hostname.IsHostnameInHostsFile(name)
			if exists {
				return
			}
			if err != nil {
				util.Warning("Could not check existence in hosts file: %v", err)
			}
			os.Exit(1)
		}
		// By default, add a host name
		err = hostname.AddHostEntry(name, dockerIP)

		if err != nil {
			util.Warning("Failed to add hosts entry %s: %v", name, err)
		}
	},
}

func init() {
	RootCmd.Flags().BoolVarP(&removeHostnameFlag, "remove", "r", false, "Remove the provided host name - ip correlation")
	RootCmd.Flags().BoolVarP(&checkHostnameFlag, "check", "c", false, "Check to see if provided hostname is already in hosts file")
}
