package cmd

import "github.com/spf13/cobra"

// DescribeCommand represents the `ddev config` command
var DescribeCommand = &cobra.Command{
	Use:   "describe",
	Short: "Get a detailed description about a running ddev site.",
	Run: func(cmd *cobra.Command, args []string) {

		//out, err := app.Describe()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// We need to override the PersistentPrerun which checks for a config.yaml in this instance,
		// since we're actually generating the config here.
	},
}

func init() {
	RootCmd.AddCommand(DescribeCommand)
}

/*func getDescribeApp(artg []string) (*platform.App, error) {
	appRoot, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}
	app := platform.NewLocalApp(appRoot)
	return app, nil
}*/
