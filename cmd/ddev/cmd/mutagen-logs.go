package cmd

import (
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
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

		ddevapp.StopMutagenDaemon()
		_ = os.Setenv("MUTAGEN_LOG_LEVEL", "trace")

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		done := make(chan bool, 1)

		go func() {
			c := exec.Command(globalconfig.GetMutagenPath(), "daemon", "run")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			err := c.Run()
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
