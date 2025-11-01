package cmd

import (
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DdevShareCommand contains the "ddev share" command
var DdevShareCommand = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "share [project]",
	Short:             "Share project on the internet via ngrok.",
	Long:              `Requires an account on ngrok.com, use the "ngrok config add-authtoken <token>" command to set up ngrok. Any ngrok flag can be added in the "ngrok_args" section of .ddev/config.yaml or via --ngrok-args.`,
	Example: `ddev share
ddev share --ngrok-args "--basic-auth username:pass1234"
ddev share --ngrok-args "--domain foo.ngrok-free.app"
ddev share myproject`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev share' or 'ddev share [projectname]'")
		}
		apps, err := getRequestedProjects(args, false)
		if err != nil {
			util.Failed("Failed to describe project(s): %v", err)
		}
		app := apps[0]

		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			util.Failed("Project is not yet running. Use 'ddev start' first.")
		}

		// Process pre-share hooks
		err = app.ProcessHooks("pre-share")
		if err != nil {
			util.Failed("Failed to process pre-share hooks: %v", err)
		}

		ngrokLoc, err := exec.LookPath("ngrok")
		if ngrokLoc == "" || err != nil {
			util.Failed("ngrok not found in path, please install it, see https://ngrok.com/download")
		}
		urls := []string{app.GetWebContainerDirectHTTPURL()}

		// Set up signal handling for SIGINT/SIGTERM to allow post-share hooks to run
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		var ngrokErr error
		var ngrokCmd *exec.Cmd
		for _, url := range urls {
			ngrokArgs := []string{"http"}
			ngrokArgs = append(ngrokArgs, url)
			if app.NgrokArgs != "" {
				ngrokArgs = append(ngrokArgs, strings.Split(app.NgrokArgs, " ")...)
			}
			if cmd.Flags().Changed("ngrok-args") {
				cmdNgrokArgs, err := cmd.Flags().GetString("ngrok-args")
				if err != nil {
					util.Failed("Unable to get --ngrok-args flag: %v", err)
				}
				ngrokArgs = append(ngrokArgs, strings.Split(cmdNgrokArgs, " ")...)
			}

			ngrokCmd = exec.Command(ngrokLoc, ngrokArgs...)
			ngrokCmd.Stdout = os.Stdout
			ngrokCmd.Stderr = os.Stderr
			err = ngrokCmd.Start()
			if err != nil {
				util.Failed("Failed to run %s %s: %v", ngrokLoc, strings.Join(ngrokArgs, " "), err)
			}
			util.Success("Running %s %s", ngrokLoc, strings.Join(ngrokArgs, " "))

			// Wait for either ngrok to exit or a signal to be received
			done := make(chan error, 1)
			go func() {
				done <- ngrokCmd.Wait()
			}()

			select {
			case ngrokErr = <-done:
				// ngrok exited on its own
			case <-sigChan:
				// Signal received, kill ngrok process
				if ngrokCmd != nil && ngrokCmd.Process != nil {
					// Ignore error from Kill() as we're already handling the exit via ngrokErr
					_ = ngrokCmd.Process.Kill()
				}
				ngrokErr = <-done
			}

			// nil result means ngrok ran and exited normally.
			// It seems to do this fine when hit by SIGTERM or SIGINT
			if ngrokErr == nil {
				break
			}

			exitErr, ok := ngrokErr.(*exec.ExitError)
			if !ok {
				// Normally we'd have an ExitError, but if not, notify
				util.Error("ngrok exited: %v", ngrokErr)
				break
			}

			exitCode := exitErr.ExitCode()
			// In the case of exitCode==1, ngrok seems to have died due to an error,
			// most likely inadequate user permissions.
			if exitCode != 1 {
				util.Error("ngrok exited: %v", exitErr)
				break
			}
			// Otherwise we'll continue and do the next url or exit
		}

		// Process post-share hooks
		err = app.ProcessHooks("post-share")
		if err != nil {
			util.Warning("Failed to process post-share hooks: %v", err)
		}

		os.Exit(0)
	},
}

func init() {
	RootCmd.AddCommand(DdevShareCommand)
	DdevShareCommand.Flags().String("ngrok-args", "", `accepts any flag from "ngrok http --help"`)
}
