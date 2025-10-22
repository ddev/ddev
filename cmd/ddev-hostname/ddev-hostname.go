package main

import (
	"os"

	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
)

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

var removeHostnameFlag bool
var checkHostnameFlag bool

// RootCmd is the ddev-hostname command
var RootCmd = &cobra.Command{
	Use:     "ddev-hostname [flags] [hostname] [ip]",
	Args:    cobra.ExactArgs(2),
	Short:   "Manage your hostfile entries.",
	Version: versionconstants.DdevVersion,
	Example: `
ddev-hostname junk.example.com 127.0.0.1
ddev-hostname -r junk.example.com 127.0.0.1
ddev-hostname --check junk.example.com 127.0.0.1
`,
	Long: `Manage your hostfile entries. Managing host names has security and usability
implications and requires elevated privileges. You may be asked for a password
to allow ddev-hostname to modify the hosts file. If you are connected to the
internet and using the domain ddev.site this is generally not necessary,
because the hosts file never gets manipulated.`,
	Run: func(_ *cobra.Command, args []string) {
		name, dockerIP := args[0], args[1]

		inHostsFile, err := isHostnameInHostsFile(name, dockerIP)
		if err != nil && checkHostnameFlag {
			printStderr("Could not check existence of '%s' in hosts file: %v\n", name, err)
		}

		// If requested, remove the provided host name and exit
		if removeHostnameFlag {
			if inHostsFile {
				if os.Getenv("DDEV_NONINTERACTIVE") == "true" || os.Getenv("CI") == "true" {
					printStderr("DDEV_NONINTERACTIVE or CI is set. Not removing the host entry.\n")
					os.Exit(0)
				}
				elevateIfNeeded()
				err := removeHostEntry(name, dockerIP)
				if err != nil {
					printStderr("Failed to remove host entry '%s' (%s): %v\n", name, dockerIP, err)
					os.Exit(1)
				} else {
					printStdout("Removed '%s' (%s) from the hosts file\n", name, dockerIP)
				}
			} else {
				printStdout("'%s' (%s) is not in the hosts file\n", name, dockerIP)
			}
			return
		}
		if checkHostnameFlag {
			if inHostsFile {
				printStdout("'%s' (%s) is in the hosts file\n", name, dockerIP)
				return
			}
			printStderr("'%s' (%s) is not in the hosts file\n", name, dockerIP)
			os.Exit(1)
		}
		// By default, add a host name
		if !inHostsFile {
			if os.Getenv("DDEV_NONINTERACTIVE") == "true" || os.Getenv("CI") == "true" {
				printStderr("DDEV_NONINTERACTIVE or CI is set. Not adding the host entry.\n")
				os.Exit(0)
			}
			elevateIfNeeded()
			err = addHostEntry(name, dockerIP)
			if err != nil {
				printStderr("Failed to add hosts entry '%s' (%s): %v\n", name, dockerIP, err)
				os.Exit(1)
			} else {
				printStdout("Added '%s' (%s) to the hosts file\n", name, dockerIP)
			}
		} else {
			printStdout("'%s' (%s) is already in the hosts file\n", name, dockerIP)
		}
	},
}

func init() {
	RootCmd.Flags().BoolVarP(&removeHostnameFlag, "remove", "r", false, "Remove the provided host name - ip correlation")
	RootCmd.Flags().BoolVarP(&checkHostnameFlag, "check", "c", false, "Check to see if provided hostname is already in hosts file")
	RootCmd.MarkFlagsMutuallyExclusive("remove", "check")
}
