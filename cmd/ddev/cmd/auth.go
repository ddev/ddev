package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// AuthCmd is the top-level "ddev auth" command
var AuthCmd = &cobra.Command{
	Use:   "auth [command]",
	Short: "A collection of authentication commands",
	Example: `ddev auth ssh
ddev auth pantheon
ddev auth ddev-live`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		util.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(AuthCmd)

	// Add additional auth. But they're currently in app. Should they be in global?
	// Try creating an app. If we get one, add it and add any items we find.
	// Or perhaps we should be loading completely different file.
	// How about calling it something other than "provider"? I guess the idea was "hosting provider"
	// maybe.

	// TODO: NewApp() will probably be a slowdown, move it out of here
	app, err := ddevapp.NewApp("", true, "")
	if err != nil {
		util.Failed("Failed to create project: %v", err)
	}

	addAuthProviders(app)
}

// addAuthProviders is simple helper function to add the shell commands
// to authenticate each provider.
func addAuthProviders(app *ddevapp.DdevApp) {
	return
	//TODO: Remove the entire auth function as it's no longer necessary

	//p, err := app.GetProvider()
	//if err != nil {
	//	util.Failed("No provider is available: %v", err)
	//}
	//p, ok := app.Provider
	//if p.AuthCommand.Command != "" {
	//	// since for-loop k,v are references, we need to get a copy of them for use in closure below.
	//	kl := k
	//	vl := v
	//	AuthCmd.AddCommand(&cobra.Command{
	//		Use:     fmt.Sprintf("%s [flags]", kl),
	//		Short:   "Authenticate " + kl + " provider",
	//		Long:    "Authenticate " + kl + " provider",
	//		Example: `ddev auth ` + kl,
	//		Run: func(cmd *cobra.Command, args []string) {
	//			if app.SiteStatus() != ddevapp.SiteRunning {
	//				util.Failed("Project %s is not running, please start it with ddev start %s", app.Name, app.Name)
	//			}
	//			err := app.ExecOnHostOrService(vl.AuthCommand.Service, vl.AuthCommand.Command)
	//			if err == nil {
	//				util.Success("Executed auth command %s on service %s", vl.AuthCommand.Command, vl.AuthCommand.Service)
	//			}
	//		}})
	//}
}
