package cmd

import (
	"fmt"
	"log"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DescribeCommand represents the `ddev config` command
var DescribeCommand = &cobra.Command{
	Use:   "describe",
	Short: "Get a detailed description of a running ddev site.",
	Run: func(cmd *cobra.Command, args []string) {
		var siteName string

		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use `ddev describe` or `ddev describe [appname]`")
		}

		if len(args) == 1 {
			siteName = args[0]
		}

		out, err := describeApp(siteName)
		if err != nil {
			log.Fatalf("Could not describe app: %v", err)
		}
		fmt.Println(out)
	},
}

// describeApp will load and describe the app specified by appName. You may leave appName blank to use the app from the current working directory.
func describeApp(appName string) (string, error) {
	var app platform.App
	var err error

	app, err = getActiveApp(appName)
	if err != nil {
		return "", err
	}

	out, err := app.Describe()
	return out, err
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}
