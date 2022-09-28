package cmd

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// MutagenLogsCmd implements the ddev mutagen logs command
var MutagenLogsCmd = &cobra.Command{
	Use:     "logs",
	Short:   "Show mutagen logs for debugging",
	Example: `"ddev mutagen logs"`,
	Run: func(cmd *cobra.Command, args []string) {
		projectName := ""
		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project-name")
		}

		if len(args) == 1 {
			projectName = args[0]
		}

		app, err := ddevapp.GetActiveApp(projectName)
		if err != nil {
			util.Failed("Failed to get active project: %v", err)
		}
		if !(app.IsMutagenEnabled()) {
			util.Warning("Mutagen is not enabled on project %s", app.Name)
			return
		}

		ddevapp.StopMutagenDaemon()
		_ = os.Setenv("MUTAGEN_LOG_LEVEL", "trace")

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		done := make(chan bool, 1)

		go func() {
			c := exec.Command(globalconfig.GetMutagenPath(), "daemon", "run")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			err = c.Run()
			if err != nil {
				util.Warning("mutagen daemon run failed with %v", err)
			}
			done <- true
		}()
		<-done

		util.Success("Completed mutagen logs, now restarting normal mutagen daemon")
		ddevapp.StartMutagenDaemon()
	},
}

func init() {
	MutagenCmd.AddCommand(MutagenLogsCmd)
	MutagenLogsCmd.Flags().Bool("verbose", false, "Show full mutagen logs")
}
