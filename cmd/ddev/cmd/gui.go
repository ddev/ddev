package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/interrupt"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/webui"
	"github.com/spf13/cobra"
)

// GUICmd opens the experimental DDEV app, or the web UI in a browser.
var GUICmd = &cobra.Command{
	Use:     "gui [project]",
	Aliases: []string{"dashboard"},
	Short:   "Open the experimental DDEV app, or the web UI in a browser",
	Long: `Open the experimental DDEV app, a graphical interface with the features of
'ddev tui': start, stop, and delete projects, follow their logs, manage
add-ons and snapshots, and toggle Xdebug.

Without the app, or with --browser, the same interface opens in the browser.
It listens on 127.0.0.1 only. The URL it prints includes an access token;
keep it private, since anyone with it can control your projects.

A project name, or the project in the current directory, is selected.`,
	Example: `ddev gui
ddev gui my-project
ddev gui --browser
ddev gui --browser --port 8765 --no-open`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Run: func(cmd *cobra.Command, args []string) {
		browser, _ := cmd.Flags().GetBool("browser")
		port, _ := cmd.Flags().GetInt("port")
		noOpen, _ := cmd.Flags().GetBool("no-open")
		desktop, _ := cmd.Flags().GetBool("desktop")

		if desktop {
			serveGUI(port, true, true, "")
			return
		}
		if !browser && (cmd.Flags().Changed("port") || noOpen) {
			util.Failed("--port and --no-open apply only with --browser")
		}
		project := ""
		if len(args) > 0 {
			app, err := ddevapp.GetActiveApp(args[0])
			if err != nil {
				util.Failed("Unable to find project %s: %v", args[0], err)
			}
			project = app.Name
		} else if app, err := ddevapp.GetActiveApp(""); err == nil {
			project = app.Name
		}

		if !browser {
			exe, reason := findDesktopApp()
			if exe != "" {
				if err := launchDesktopApp(exe, project); err != nil {
					util.Failed("Unable to open the DDEV app %s: %v", exe, err)
				}
				output.UserOut.Printf("Opened the DDEV app (experimental). Use 'ddev gui --browser' for the web UI.")
				return
			}
			util.Warning("%s, so the DDEV web UI opens in your browser instead.", reason)
		}
		serveGUI(port, noOpen, false, project)
	},
}

// serveGUI serves the web UI until Ctrl+C. With desktop, it is the server
// that the DDEV app shows in its window, and it stops when stdin closes.
func serveGUI(port int, noOpen bool, desktop bool, project string) {
	// Hooks print to this terminal; the subcommands the web UI runs, such
	// as start, still run theirs.
	ddevapp.SkipHooks = true
	srv, err := webui.New(port)
	if err != nil {
		util.Failed("Unable to start the DDEV web UI: %v", err)
	}
	if desktop {
		srv.SetDesktop()
		srv.StopWithStdin()
	}
	unregister := interrupt.OnInterrupt(func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	defer unregister()

	launch := srv.URL()
	if project != "" {
		launch += "#project=" + url.QueryEscape(project)
	}
	output.UserOut.Printf("DDEV web UI (experimental) is at\n\n    %s\n\nKeep this URL private: its token gives control of your projects. Press Ctrl+C to stop.", launch)
	if !noOpen {
		if err := webui.OpenBrowser(launch); err != nil {
			util.Warning("Unable to open a browser: %v", err)
		}
	}
	if err := srv.Serve(); err != nil {
		util.Failed("DDEV web UI failed: %v", err)
	}
}

// findDesktopApp returns the DDEV app's executable, or why there is none.
// $DDEV_DESKTOP_APP names it; otherwise the app is looked for next to this
// ddev, where packages and 'ddev utility download-ddev' put it, then where
// installers put it.
func findDesktopApp() (exe string, reason string) {
	if e := os.Getenv("DDEV_DESKTOP_APP"); e != "" {
		if nodeps.IsMacOS() && filepath.Ext(e) == ".app" {
			e = filepath.Join(e, "Contents", "MacOS", "ddev-desktop")
		}
		if !fileutil.FileExists(e) {
			return "", fmt.Sprintf("DDEV_DESKTOP_APP names %s, which does not exist", e)
		}
		return e, ""
	}
	switch {
	case nodeps.IsWSL2():
		return "", "The DDEV app does not run inside WSL2"
	case os.Getenv("SSH_CONNECTION") != "":
		return "", "This is an SSH session"
	case nodeps.IsLinux() && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "":
		return "", "There is no graphical session (DISPLAY and WAYLAND_DISPLAY are unset)"
	}
	dir := ""
	if self, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(self); err == nil {
			self = resolved
		}
		dir = filepath.Dir(self)
	}
	var candidates []string
	if nodeps.IsMacOS() {
		home, _ := os.UserHomeDir()
		for _, d := range []string{dir, "/Applications", filepath.Join(home, "Applications")} {
			candidates = append(candidates, filepath.Join(d, "DDEV.app", "Contents", "MacOS", "ddev-desktop"))
		}
	} else {
		name := "ddev-desktop"
		if nodeps.IsWindows() {
			name += ".exe"
		}
		candidates = append(candidates, filepath.Join(dir, name))
		if p, err := exec.LookPath(name); err == nil {
			candidates = append(candidates, p)
		}
	}
	for _, c := range candidates {
		if fileutil.FileExists(c) {
			return c, ""
		}
	}
	return "", fmt.Sprintf("The DDEV app was not found next to %s", filepath.Join(dir, "ddev"))
}

// launchDesktopApp starts the DDEV app, which runs this ddev. A running app
// takes the arguments of the new one, shows its window, and selects the
// project, so the new one exits.
func launchDesktopApp(exe string, project string) error {
	var args []string
	if project != "" {
		args = append(args, "--project", project)
	}
	c := exec.Command(exe, args...)
	if self, err := os.Executable(); err == nil {
		c.Env = append(os.Environ(), "DDEV_BINARY="+self)
	}
	// Its own process group keeps Ctrl+C in this terminal from reaching it.
	setProcessGroupAttr(c)
	if err := c.Start(); err != nil {
		return err
	}
	return c.Process.Release()
}

func registerGUICmd() {
	GUICmd.Flags().Bool("browser", false, "Open the web UI in the browser instead of the DDEV app")
	GUICmd.Flags().Int("port", 0, "With --browser, the port to listen on at 127.0.0.1; a free port is chosen when 0")
	GUICmd.Flags().Bool("no-open", false, "With --browser, print the URL without opening a browser")
	// The DDEV app runs `ddev gui --desktop` and shows it in its window.
	GUICmd.Flags().Bool("desktop", false, "Serve the web UI for the DDEV app's window")
	_ = GUICmd.Flags().MarkHidden("desktop")
	RootCmd.AddCommand(GUICmd)
}
