package cmd

import (
	"github.com/drud/ddev/pkg/exec"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
)

// DebugMutagenCmd implements the ddev debug mutagen command
var DebugMutagenCmd = &cobra.Command{
	Use:   "mutagen",
	Short: "Allows access to any mutagen command",
	Long:  "This simply passes through any mutagen command to the embedded mutagen itself. See Mutagen docs at https://mutagen.io/documentation/introduction",
	Example: `ddev debug mutagen sync list
ddev debug mutagen daemon stop
ddev debug mutagen
`,
	Run: func(cmd *cobra.Command, args []string) {
		mutagenPath := globalconfig.GetMutagenPath()
		_, err := os.Stat(mutagenPath)
		if err != nil {
			util.Warning("mutagen does not seem to be set up in %s, not executing command", mutagenPath)
			return
		}
		out, err := exec.RunHostCommand(mutagenPath, args...)
		output.UserOut.Printf(out)
		if err != nil {
			util.Failed("Error running '%s %v': %v", globalconfig.GetMutagenPath(), args, err)
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugMutagenCmd)
}
