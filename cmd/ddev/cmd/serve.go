package cmd

import (
	"context"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/interrupt"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/webui"
	"github.com/spf13/cobra"
)

// ServeCmd serves the experimental browser dashboard.
var ServeCmd = &cobra.Command{
	Use:         "serve",
	Annotations: map[string]string{NoDockerCommand: "true"},
	Short:       "Serve an experimental dashboard in the browser",
	Long: `Serve an experimental browser dashboard for managing DDEV projects, with
the features of 'ddev tui': start, stop, and delete projects, follow their
logs, manage add-ons and snapshots, and toggle Xdebug.

The dashboard listens on 127.0.0.1 only. The URL it prints includes an access
token; keep it private, since anyone with it can control your projects.`,
	Example: `ddev serve
ddev serve --port 8765
ddev serve --no-open`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		port, _ := cmd.Flags().GetInt("port")
		noOpen, _ := cmd.Flags().GetBool("no-open")

		// Hooks print to this terminal; the subcommands the dashboard runs,
		// such as start, still run theirs.
		ddevapp.SkipHooks = true
		srv, err := webui.New(port)
		if err != nil {
			util.Failed("Unable to start the dashboard: %v", err)
		}
		unregister := interrupt.OnInterrupt(func(ctx context.Context) {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		})
		defer unregister()

		url := srv.URL()
		output.UserOut.Printf("DDEV dashboard (experimental) is at\n\n    %s\n\nKeep this URL private: its token gives control of your projects. Press Ctrl+C to stop.", url)
		if !noOpen {
			if err := webui.OpenBrowser(url); err != nil {
				util.Warning("Unable to open a browser: %v", err)
			}
		}
		if err := srv.Serve(); err != nil {
			util.Failed("Dashboard failed: %v", err)
		}
	},
}

func registerServeCmd() {
	ServeCmd.Flags().Int("port", 0, "Port to listen on at 127.0.0.1; a free port is chosen when 0")
	ServeCmd.Flags().Bool("no-open", false, "Print the URL without opening a browser")
	RootCmd.AddCommand(ServeCmd)
}
