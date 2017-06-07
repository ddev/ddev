package cmd

import (
	"fmt"
	"log"

	"github.com/drud/ddev/pkg/dockerutil"
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

// describeApp will load and describe the app specified by appName. You may leave appName blank to use the app from the current working directory.
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
			"com.ddev.site-name":         appName,
			"com.docker.compose.service": "web",
		}

		webContainer, err := dockerutil.FindContainerByLabels(labels)
		if err != nil {
			return "", err
		}

		dir, ok := webContainer.Labels["com.ddev.approot"]
		if !ok {
			return "", fmt.Errorf("could not find webroot on container: %s", dockerutil.ContainerName(webContainer))
		}

		app, err = platform.GetPluginApp(plugin)
		if err != nil {
			log.Fatalf("Could not find application type %s: %v", plugin, err)
		}

		err = app.Init(dir)
		if err != nil {
			return "", err
		}

	}
	out, err := app.Describe()
	return out, err
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}
