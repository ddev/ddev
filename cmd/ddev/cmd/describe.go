package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/drud/ddev/pkg/plugins/platform"
	"github.com/spf13/cobra"
)

// DescribeCommand represents the `ddev config` command
var DescribeCommand = &cobra.Command{
	Use:   "describe",
	Short: "Get a detailed description of a running ddev site.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			log.Fatal("Too many arguments detected. Please use `ddev describe` or `ddev describe [appname]`")
		}

		appName := ""

		if len(args) == 1 {
			appName = args[0]
		}

		out, err := describeApp(appName)
		if err != nil {
			log.Fatalf("Could not describe app: %v", err)
		}

		fmt.Println(out)
	},
}

func describeApp(appName string) (string, error) {
	var app platform.App
	var err error

	if appName == "" {
		app, err = getActiveApp()
		if err != nil {
			return "", err
		}
	} else {
		labels := map[string]string{
			"com.ddev.site-name":      appName,
			"com.ddev.container-type": "web",
		}

		webContainer, err := platform.FindContainerByLabels(labels)
		if err != nil {
			return "", err
		}

		if dir, ok := webContainer.Labels["com.ddev.approot"]; ok {
			if err != nil {
				return "", err
			}
			app = platform.PluginMap[strings.ToLower(plugin)]
			err = app.Init(dir)
		}
	}

	out, err := app.Describe()
	return out, err
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}
