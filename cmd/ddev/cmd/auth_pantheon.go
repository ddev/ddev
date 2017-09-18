package cmd

import (
	"path/filepath"

	"github.com/drud/ddev/pkg/util"
	"github.com/drud/go-pantheon/pkg/pantheon"
	gohomedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

// PantheonAuthCommand represents the `ddev auth-pantheon` command
var PantheonAuthCommand = &cobra.Command{
	Use:   "auth-pantheon [token]",
	Short: "Provide a machine token for the global pantheon auth.",
	Long:  "Configure global machine token for pantheon authentication. See https://pantheon.io/docs/machine-tokens/ for instructions on creating a token.",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			util.Failed("You must provide a Pantheon machine token, e.g. `ddev auth-pantheon [token]`. See https://pantheon.io/docs/machine-tokens/ for instructions on creating a token.")
		}
		if len(args) != 1 {
			util.Failed("Too many arguments detected. Please provide only your Pantheon Machine token., e.g. `ddev auth-pantheon [token]`. See https://pantheon.io/docs/machine-tokens/ for instructions on creating a token.")
		}
		userDir, err := gohomedir.Dir()
		util.CheckErr(err)
		sessionLocation := filepath.Join(userDir, ".ddev", "pantheonconfig.json")

		session := pantheon.NewAuthSession(args[0])
		err = session.Auth()
		if err != nil {
			util.Failed("Could not authenticate with pantheon: %v", err)
		}

		err = session.Write(sessionLocation)
		if err != nil {
			util.Failed("Failed session.Write(), err=%v", err)
		}
		util.Success("Authentication successful!\nYou may now use the `ddev config pantheon` command when configuring sites!")
	},
}

func init() {
	RootCmd.AddCommand(PantheonAuthCommand)
}
