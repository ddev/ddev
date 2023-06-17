package cmd

import (
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
)

// DebugMutagenCmd implements the ddev debug mutagen command
var DebugMutagenCmd = &cobra.Command{
	Use:   "mutagen",
	Short: "Allows access to any mutagen command",
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},

	Long: "This simply passes through any mutagen command to the embedded mutagen itself. See Mutagen docs at https://mutagen.io/documentation/introduction",
	Example: `ddev debug mutagen sync list
ddev debug mutagen daemon stop
ddev debug mutagen
ddev d mutagen sync list
`,
	Run: func(cmd *cobra.Command, args []string) {
		mutagenPath := globalconfig.GetMutagenPath()
		_, err := os.Stat(mutagenPath)
		if err != nil {
			util.Warning("mutagen does not seem to be set up in %s, not executing command", mutagenPath)
			return
		}
		out, err := exec.RunHostCommand(mutagenPath, os.Args[3:]...)
		output.UserOut.Printf(out)
		if err != nil {
			util.Failed("Error running '%s %v': %v", globalconfig.GetMutagenPath(), args, err)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugMutagenCmd)
}
